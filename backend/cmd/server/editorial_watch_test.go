package main

import (
	"context"
	"sync"
	"testing"
	"time"

	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/playout"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// watchTest runs api.watch in the background and collects what it sends.
type watchTest struct {
	events chan *apiv1.EditorialSessionEvent
	done   chan error
	cancel context.CancelFunc
	once   sync.Once
	err    error
}

// stop closes the tab and returns what watch returned. Safe to call twice.
func (w *watchTest) stop() error {
	w.once.Do(func() {
		w.cancel()
		w.err = <-w.done
	})
	return w.err
}

func startWatch(t *testing.T, api *EditorialAPI, sessionID, clientID, email string) *watchTest {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	w := &watchTest{events: make(chan *apiv1.EditorialSessionEvent, 64), done: make(chan error, 1), cancel: cancel}
	go func() {
		w.done <- api.watch(ctx, sessionID, clientID, email, true, func(ev *apiv1.EditorialSessionEvent) error {
			// Like a real stream, a send gives up once the client is gone,
			// so a test that stops reading can't wedge the watcher.
			select {
			case w.events <- ev:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	t.Cleanup(func() { _ = w.stop() })
	return w
}

// next waits for the next event that matches keep.
func (w *watchTest) next(t *testing.T, keep func(*apiv1.EditorialSessionEvent) bool) *apiv1.EditorialSessionEvent {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case ev := <-w.events:
			if keep(ev) {
				return ev
			}
		case <-timeout:
			t.Fatal("no matching event")
			return nil
		}
	}
}

func hasSession(ev *apiv1.EditorialSessionEvent) bool { return ev.Session != nil }

// The marking/commenting pair: one tab marks, the other sees the marker
// arrive; the other comments, the first sees the comment.
func TestWatchSendsChangesToOtherTabs(t *testing.T) {
	api := newTestEditorialAPI(t)
	ctx := context.Background()
	created, err := api.CreateEditorialSession(ctx, connect.NewRequest(&apiv1.CreateEditorialSessionRequest{Live: true, PlayoutEventId: "1013"}))
	require.NoError(t, err)
	id := created.Msg.Id

	marker := startWatch(t, api, id, "tab-marker", "editor@bcc.media")
	commenter := startWatch(t, api, id, "tab-commenter", "editor@bcc.media")
	first := commenter.next(t, hasSession)
	assert.Empty(t, first.Session.Markers)

	added, err := api.AddEditorialMarker(ctx, connect.NewRequest(&apiv1.AddEditorialMarkerRequest{SessionId: id, At: timestamppb.Now()}))
	require.NoError(t, err)
	ev := commenter.next(t, func(ev *apiv1.EditorialSessionEvent) bool { return hasSession(ev) && len(ev.Session.Markers) == 1 })
	assert.Equal(t, added.Msg.Id, ev.Session.Markers[0].Id)

	_, err = api.SetEditorialComment(ctx, connect.NewRequest(&apiv1.SetEditorialCommentRequest{SessionId: id, MarkerId: added.Msg.Id, Comment: "Sjekk navn"}))
	require.NoError(t, err)
	ev = marker.next(t, func(ev *apiv1.EditorialSessionEvent) bool {
		return hasSession(ev) && len(ev.Session.Markers) == 1 && ev.Session.Markers[0].Comment == "Sjekk navn"
	})
	assert.Equal(t, "Sjekk navn", ev.Session.Markers[0].Comment)
}

func TestWatchPresence(t *testing.T) {
	api := newTestEditorialAPI(t)
	ctx := context.Background()
	created, err := api.CreateEditorialSession(ctx, connect.NewRequest(&apiv1.CreateEditorialSessionRequest{Live: true, PlayoutEventId: "1013"}))
	require.NoError(t, err)
	id := created.Msg.Id

	a := startWatch(t, api, id, "tab-a", "editor@bcc.media")
	a.next(t, hasSession)
	b := startWatch(t, api, id, "tab-b", "editor@bcc.media")

	ev := a.next(t, func(ev *apiv1.EditorialSessionEvent) bool { return len(ev.Presence) == 2 })
	assert.Equal(t, []string{"tab-a", "tab-b"}, []string{ev.Presence[0].ClientId, ev.Presence[1].ClientId})

	_, err = api.SetEditorialPresence(ctx, connect.NewRequest(&apiv1.SetEditorialPresenceRequest{SessionId: id, ClientId: "tab-b", Activity: "commenting", MarkerId: "m1"}))
	require.NoError(t, err)
	ev = a.next(t, func(ev *apiv1.EditorialSessionEvent) bool {
		return len(ev.Presence) == 2 && ev.Presence[1].Activity == "commenting"
	})
	assert.Equal(t, "m1", ev.Presence[1].MarkerId)

	_, err = api.SetEditorialPresence(ctx, connect.NewRequest(&apiv1.SetEditorialPresenceRequest{SessionId: id, ClientId: "tab-b", Activity: "dancing"}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	// Closing a tab removes it.
	require.NoError(t, b.stop())
	a.next(t, func(ev *apiv1.EditorialSessionEvent) bool { return len(ev.Presence) == 1 })
}

// A tab can't set someone else's presence: the email must match its stream.
func TestPresenceIsPerUser(t *testing.T) {
	hub := newEditorialHub()
	w := hub.join("s1", "tab", "kari@bcc.media", false)
	defer hub.leave(w)
	assert.False(t, hub.setPresence("s1", "tab", "ola@bcc.media", "commenting", "m1"))
	assert.True(t, hub.setPresence("s1", "tab", "kari@bcc.media", "commenting", "m1"))
	assert.Equal(t, "commenting", hub.presence("s1")[0].Activity)
}

func TestWatchEndsWhenSessionIsDeleted(t *testing.T) {
	api := newTestEditorialAPI(t)
	ctx := context.Background()
	created, err := api.CreateEditorialSession(ctx, connect.NewRequest(&apiv1.CreateEditorialSessionRequest{Live: true, PlayoutEventId: "1013"}))
	require.NoError(t, err)
	w := startWatch(t, api, created.Msg.Id, "tab", "editor@bcc.media")
	w.next(t, hasSession)

	_, err = api.DeleteEditorialSession(ctx, connect.NewRequest(&apiv1.DeleteEditorialSessionRequest{Id: created.Msg.Id}))
	require.NoError(t, err)
	w.next(t, func(ev *apiv1.EditorialSessionEvent) bool { return ev.Deleted })
	// The stream ends by itself; stop only collects its result.
	select {
	case err := <-w.done:
		assert.NoError(t, err)
		w.once.Do(func() {})
	case <-time.After(2 * time.Second):
		t.Fatal("stream didn't end")
	}
}

func TestWatchKeepAlive(t *testing.T) {
	api := newTestEditorialAPI(t)
	old := watchKeepAlive
	watchKeepAlive = 20 * time.Millisecond
	t.Cleanup(func() { watchKeepAlive = old })
	created, err := api.CreateEditorialSession(context.Background(), connect.NewRequest(&apiv1.CreateEditorialSessionRequest{Live: true, PlayoutEventId: "1013"}))
	require.NoError(t, err)

	w := startWatch(t, api, created.Msg.Id, "tab", "editor@bcc.media")
	w.next(t, hasSession)
	w.next(t, func(ev *apiv1.EditorialSessionEvent) bool {
		return ev.Session == nil && ev.Presence == nil && !ev.Deleted
	})
}

func TestSyncLock(t *testing.T) {
	hub := newEditorialHub()
	assert.True(t, hub.maySync("s1", "anyone"), "nobody holds it while nobody watches")

	reviewer := hub.join("s1", "tab-reviewer", "rev@bcc.media", false)
	assert.Empty(t, hub.syncClient("s1"), "a reviewer's tab never holds it")
	first := hub.join("s1", "tab-1", "kari@bcc.media", true)
	second := hub.join("s1", "tab-2", "ola@bcc.media", true)
	assert.Equal(t, "tab-1", hub.syncClient("s1"), "the first live tab holds it")
	assert.True(t, hub.maySync("s1", "tab-1"))
	assert.False(t, hub.maySync("s1", "tab-2"))

	assert.False(t, hub.takeSync("s1", "tab-2", "kari@bcc.media"), "only the tab's own user can take over")
	assert.False(t, hub.takeSync("s1", "tab-reviewer", "rev@bcc.media"), "a reviewer can't take it")
	assert.True(t, hub.takeSync("s1", "tab-2", "ola@bcc.media"))
	assert.Equal(t, "tab-2", hub.syncClient("s1"))

	hub.leave(second)
	assert.Equal(t, "tab-1", hub.syncClient("s1"), "passes on when the holder leaves")
	hub.leave(first)
	assert.Empty(t, hub.syncClient("s1"), "released when no live tab is left")
	hub.leave(reviewer)
}

func TestSyncRefusedForOtherTabs(t *testing.T) {
	api := newTestEditorialAPI(t)
	api.playout = fakePlayout{manifest: &playout.Manifest{}}
	ctx := context.Background()
	created, err := api.CreateEditorialSession(ctx, connect.NewRequest(&apiv1.CreateEditorialSessionRequest{Live: true, PlayoutEventId: "1013"}))
	require.NoError(t, err)
	id := created.Msg.Id

	holder := startWatch(t, api, id, "tab-1", "editor@bcc.media")
	holder.next(t, func(ev *apiv1.EditorialSessionEvent) bool { return ev.SyncClientId == "tab-1" })
	startWatch(t, api, id, "tab-2", "editor@bcc.media")

	_, err = api.SyncEditorialSessionFromPlayout(ctx, connect.NewRequest(&apiv1.SyncEditorialSessionFromPlayoutRequest{Id: id, ClientId: "tab-2"}))
	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	_, err = api.SyncEditorialSessionFromPlayout(ctx, connect.NewRequest(&apiv1.SyncEditorialSessionFromPlayoutRequest{Id: id, ClientId: "tab-1"}))
	assert.NoError(t, err)

	_, err = api.TakeEditorialSync(ctx, connect.NewRequest(&apiv1.TakeEditorialSyncRequest{SessionId: id, ClientId: "tab-2"}))
	require.NoError(t, err)
	holder.next(t, func(ev *apiv1.EditorialSessionEvent) bool { return ev.SyncClientId == "tab-2" })
	_, err = api.SyncEditorialSessionFromPlayout(ctx, connect.NewRequest(&apiv1.SyncEditorialSessionFromPlayoutRequest{Id: id, ClientId: "tab-2"}))
	assert.NoError(t, err)
}
