package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Several people work on one session at once, typically someone marking a
// live meeting and someone commenting on a phone. Every open page watches its
// session over a server stream; any change wakes the watchers, who each send
// their client the session as it now is. Sessions are small, so sending the
// whole thing keeps clients simple and correct.
//
// The hub lives in memory: the server is a single instance (the editorial
// database is a local SQLite file), so there is no one else to tell.

// watchKeepAlive is how often a quiet stream sends an empty event, well under
// the idle timeouts of proxies in front of the server.
var watchKeepAlive = 20 * time.Second

type editorialHub struct {
	mu       sync.Mutex
	sessions map[string]map[*watcher]struct{}
	// The tab that runs the Playout sync for each session, so a session with
	// several live users open syncs once. Held by a tab that may sync; passed
	// on, in join order, when it leaves.
	syncOwner map[string]*watcher
	joined    int
}

func newEditorialHub() *editorialHub {
	return &editorialHub{sessions: map[string]map[*watcher]struct{}{}, syncOwner: map[string]*watcher{}}
}

// watcher is one open stream. Its wake channels hold at most one pending
// signal, so a burst of changes coalesces into one send.
type watcher struct {
	sessionID string
	// the user has the live role, so this tab may run the Playout sync
	canSync   bool
	order     int
	presence  *apiv1.EditorialPresence
	changed   chan struct{}
	presenceC chan struct{}
}

func signal(c chan struct{}) {
	select {
	case c <- struct{}{}:
	default:
	}
}

func (h *editorialHub) join(sessionID, clientID, email string, canSync bool) *watcher {
	w := &watcher{
		sessionID: sessionID,
		canSync:   canSync,
		presence:  &apiv1.EditorialPresence{ClientId: clientID, Email: email, Activity: "viewing"},
		changed:   make(chan struct{}, 1),
		presenceC: make(chan struct{}, 1),
	}
	h.mu.Lock()
	if h.sessions[sessionID] == nil {
		h.sessions[sessionID] = map[*watcher]struct{}{}
	}
	h.joined++
	w.order = h.joined
	h.sessions[sessionID][w] = struct{}{}
	if canSync && h.syncOwner[sessionID] == nil {
		h.syncOwner[sessionID] = w
	}
	h.mu.Unlock()
	h.presenceChanged(sessionID)
	return w
}

func (h *editorialHub) leave(w *watcher) {
	h.mu.Lock()
	delete(h.sessions[w.sessionID], w)
	if len(h.sessions[w.sessionID]) == 0 {
		delete(h.sessions, w.sessionID)
	}
	if h.syncOwner[w.sessionID] == w {
		// The longest-open tab that may sync takes over.
		var next *watcher
		for other := range h.sessions[w.sessionID] {
			if other.canSync && (next == nil || other.order < next.order) {
				next = other
			}
		}
		if next != nil {
			h.syncOwner[w.sessionID] = next
		} else {
			delete(h.syncOwner, w.sessionID)
		}
	}
	h.mu.Unlock()
	h.presenceChanged(w.sessionID)
}

// changed tells everyone watching the session to send it again. Safe on a
// nil hub (tests build EditorialAPI without one).
func (h *editorialHub) changed(sessionID string) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for w := range h.sessions[sessionID] {
		signal(w.changed)
	}
}

func (h *editorialHub) presenceChanged(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for w := range h.sessions[sessionID] {
		signal(w.presenceC)
	}
}

// setPresence updates what one tab is doing. Only that tab's own user may
// set it: the email must match the stream that joined.
func (h *editorialHub) setPresence(sessionID, clientID, email, activity, markerID string) bool {
	h.mu.Lock()
	found := false
	for w := range h.sessions[sessionID] {
		if w.presence.ClientId == clientID && w.presence.Email == email {
			w.presence = &apiv1.EditorialPresence{ClientId: clientID, Email: email, Activity: activity, MarkerId: markerID}
			found = true
		}
	}
	h.mu.Unlock()
	if found {
		h.presenceChanged(sessionID)
	}
	return found
}

// presence lists everyone watching the session, in a stable order.
func (h *editorialHub) presence(sessionID string) []*apiv1.EditorialPresence {
	h.mu.Lock()
	out := make([]*apiv1.EditorialPresence, 0, len(h.sessions[sessionID]))
	for w := range h.sessions[sessionID] {
		out = append(out, w.presence)
	}
	h.mu.Unlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].Email != out[j].Email {
			return out[i].Email < out[j].Email
		}
		return out[i].ClientId < out[j].ClientId
	})
	return out
}

// syncClient is the tab holding the session's sync lock, or "".
func (h *editorialHub) syncClient(sessionID string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if w := h.syncOwner[sessionID]; w != nil {
		return w.presence.ClientId
	}
	return ""
}

// maySync reports whether a tab may run the session's Playout sync: it holds
// the lock, or no open tab does (nobody is watching, or the stream is down
// for everyone).
func (h *editorialHub) maySync(sessionID, clientID string) bool {
	if h == nil {
		return true
	}
	owner := h.syncClient(sessionID)
	return owner == "" || owner == clientID
}

// takeSync moves the lock to one of the user's own tabs that may sync.
func (h *editorialHub) takeSync(sessionID, clientID, email string) bool {
	h.mu.Lock()
	found := false
	for w := range h.sessions[sessionID] {
		if w.presence.ClientId == clientID && w.presence.Email == email && w.canSync {
			h.syncOwner[sessionID] = w
			found = true
		}
	}
	h.mu.Unlock()
	if found {
		h.presenceChanged(sessionID)
	}
	return found
}

var presenceActivities = map[string]bool{"viewing": true, "commenting": true, "editing": true}

func (e EditorialAPI) WatchEditorialSession(ctx context.Context, req *connect.Request[apiv1.WatchEditorialSessionRequest], stream *connect.ServerStream[apiv1.EditorialSessionEvent]) error {
	email, err := requireEditorial(req, apiv1.EditorialReviewer)
	if err != nil {
		return err
	}
	if e.hub == nil {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("live updates are not available"))
	}
	if req.Msg.GetId() == "" || req.Msg.GetClientId() == "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing id or client_id"))
	}
	canSync := PermissionsForEmail(email).EditorialRole() >= apiv1.EditorialLive
	return e.watch(ctx, req.Msg.GetId(), req.Msg.GetClientId(), email, canSync, stream.Send)
}

// watch sends the session and its presence until ctx ends or the session is
// deleted. send is the stream's Send. A closed tab ends it without error,
// whatever it was doing at the time.
func (e EditorialAPI) watch(ctx context.Context, sessionID, clientID, email string, canSync bool, send func(*apiv1.EditorialSessionEvent) error) error {
	err := e.watchUntilDone(ctx, sessionID, clientID, email, canSync, send)
	if ctx.Err() != nil {
		return nil
	}
	return err
}

func (e EditorialAPI) watchUntilDone(ctx context.Context, sessionID, clientID, email string, canSync bool, send func(*apiv1.EditorialSessionEvent) error) error {
	w := e.hub.join(sessionID, clientID, email, canSync)
	defer e.hub.leave(w)

	// sendSession sends the session as it is now; done reports the session
	// was deleted and the stream should end.
	sendSession := func() (done bool, err error) {
		sess, err := e.loadSession(ctx, sessionID)
		if errors.Is(err, errEditorialNotFound) {
			return true, send(&apiv1.EditorialSessionEvent{Deleted: true})
		}
		if err != nil {
			return false, connect.NewError(connect.CodeInternal, err)
		}
		// So open pages switch video when an admin changes the asset.
		if sess.VXID != "" {
			sess.PreviewUrl = editorialPreviewPath(sess.Id, sess.VXID)
		}
		return false, send(&apiv1.EditorialSessionEvent{Session: sess, Presence: e.hub.presence(sessionID), SyncClientId: e.hub.syncClient(sessionID)})
	}

	if done, err := sendSession(); done || err != nil {
		return err
	}
	keepAlive := time.NewTicker(watchKeepAlive)
	defer keepAlive.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-w.changed:
			if done, err := sendSession(); done || err != nil {
				return err
			}
		case <-w.presenceC:
			if err := send(&apiv1.EditorialSessionEvent{Presence: e.hub.presence(sessionID), SyncClientId: e.hub.syncClient(sessionID)}); err != nil {
				return err
			}
		case <-keepAlive.C:
			if err := send(&apiv1.EditorialSessionEvent{}); err != nil {
				return err
			}
		}
	}
}

func (e EditorialAPI) SetEditorialPresence(ctx context.Context, req *connect.Request[apiv1.SetEditorialPresenceRequest]) (*connect.Response[apiv1.Void], error) {
	email, err := requireEditorial(req, apiv1.EditorialReviewer)
	if err != nil {
		return nil, err
	}
	if !presenceActivities[req.Msg.GetActivity()] {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown activity %q", req.Msg.GetActivity()))
	}
	if e.hub == nil {
		return connect.NewResponse(&apiv1.Void{}), nil
	}
	// A tab whose stream isn't connected has no presence to update; that's
	// not an error the user can act on.
	e.hub.setPresence(req.Msg.GetSessionId(), req.Msg.GetClientId(), email, req.Msg.GetActivity(), req.Msg.GetMarkerId())
	return connect.NewResponse(&apiv1.Void{}), nil
}

func (e EditorialAPI) TakeEditorialSync(ctx context.Context, req *connect.Request[apiv1.TakeEditorialSyncRequest]) (*connect.Response[apiv1.Void], error) {
	email, err := requireEditorial(req, apiv1.EditorialLive)
	if err != nil {
		return nil, err
	}
	if e.hub == nil || !e.hub.takeSync(req.Msg.GetSessionId(), req.Msg.GetClientId(), email) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("this tab isn't connected to the session"))
	}
	return connect.NewResponse(&apiv1.Void{}), nil
}

// GetServerTime reports the server's clock. A page compares it with its own
// to see how far the computer's clock is off: markers are stamped with the
// computer's clock, Playout's entries with Playout's.
func (e EditorialAPI) GetServerTime(ctx context.Context, req *connect.Request[apiv1.Void]) (*connect.Response[apiv1.ServerTime], error) {
	if _, err := requireEditorial(req, apiv1.EditorialReviewer); err != nil {
		return nil, err
	}
	return connect.NewResponse(&apiv1.ServerTime{Now: timestamppb.Now()}), nil
}
