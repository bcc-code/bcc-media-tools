package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/editorial"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// approvalUndoTTL is how long an undo token stays usable. Longer than the
// toast that offers Undo, whose countdown pauses while it is hovered or
// focused.
const approvalUndoTTL = 10 * time.Minute

// approvalState is one approval as stored: approved or not, and who approved
// it and when (empty while not approved).
type approvalState struct {
	approved bool
	by       string
	at       int64
}

func sessionApproval(s editorial.Session, kind apiv1.EditorialApprovalKind) approvalState {
	if kind == apiv1.EditorialApprovalKind_EDITORIAL_APPROVAL_KIND_TECHNICAL {
		return approvalState{s.TechnicalApproved, s.TechnicalApprovedBy, s.TechnicalApprovedAt}
	}
	return approvalState{s.RedactionalApproved, s.RedactionalApprovedBy, s.RedactionalApprovedAt}
}

// validApprovalKind reports whether kind names an approval a session has.
func validApprovalKind(kind apiv1.EditorialApprovalKind) bool {
	return kind == apiv1.EditorialApprovalKind_EDITORIAL_APPROVAL_KIND_REDACTIONAL ||
		kind == apiv1.EditorialApprovalKind_EDITORIAL_APPROVAL_KIND_TECHNICAL
}

// swapApproval sets one approval to `to`, but only while it is still `was`.
// Returns the rows changed: 0 when the session is gone or the approval has
// changed since the caller read it.
func (e EditorialAPI) swapApproval(ctx context.Context, sessionID string, kind apiv1.EditorialApprovalKind, was, to approvalState, now int64) (int64, error) {
	if kind == apiv1.EditorialApprovalKind_EDITORIAL_APPROVAL_KIND_TECHNICAL {
		return e.queries.SwapTechnicalApproval(ctx, editorial.SwapTechnicalApprovalParams{
			Approved:      to.approved,
			ApprovedBy:    to.by,
			ApprovedAt:    to.at,
			UpdatedAt:     now,
			ID:            sessionID,
			WasApproved:   was.approved,
			WasApprovedBy: was.by,
			WasApprovedAt: was.at,
		})
	}
	return e.queries.SwapRedactionalApproval(ctx, editorial.SwapRedactionalApprovalParams{
		Approved:      to.approved,
		ApprovedBy:    to.by,
		ApprovedAt:    to.at,
		UpdatedAt:     now,
		ID:            sessionID,
		WasApproved:   was.approved,
		WasApprovedBy: was.by,
		WasApprovedAt: was.at,
	})
}

// approvalUndo is what one SetEditorialApproval call replaced, so Undo can put
// back the original approver and time instead of approving as the caller.
type approvalUndo struct {
	sessionID string
	kind      apiv1.EditorialApprovalKind
	// who made the change; only they may undo it
	email string
	// before and after the change; Undo applies only while it is still after
	before, after approvalState
	expires       time.Time
}

// approvalUndoStore keeps undo tokens in memory. A restart forgets them,
// which only means an Undo offered just before it no longer works.
type approvalUndoStore struct {
	mu      sync.Mutex
	entries map[string]approvalUndo
	now     func() time.Time
}

func newApprovalUndoStore() *approvalUndoStore {
	return &approvalUndoStore{entries: map[string]approvalUndo{}, now: time.Now}
}

// put stores u and returns its token. Expired entries are dropped on the way.
func (s *approvalUndoStore) put(u approvalUndo) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	for token, e := range s.entries {
		if now.After(e.expires) {
			delete(s.entries, token)
		}
	}
	u.expires = now.Add(approvalUndoTTL)
	token := uuid.NewString()
	s.entries[token] = u
	return token
}

// take returns and forgets the entry for token, if it exists, hasn't expired
// and was made by email for sessionID. Someone else's token stays usable by
// its owner.
func (s *approvalUndoStore) take(token, email, sessionID string) (approvalUndo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.entries[token]
	if !ok || u.email != email || u.sessionID != sessionID {
		return approvalUndo{}, fmt.Errorf("unknown undo token")
	}
	delete(s.entries, token)
	if s.now().After(u.expires) {
		return approvalUndo{}, fmt.Errorf("undo token expired")
	}
	return u, nil
}
