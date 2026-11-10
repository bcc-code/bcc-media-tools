package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"context"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const technical = apiv1.EditorialApprovalKind_EDITORIAL_APPROVAL_KIND_TECHNICAL

// setTestApproval sets one approval as the current caller and returns the
// undo token.
func setTestApproval(t *testing.T, api *EditorialAPI, sessionID string, approved bool) string {
	t.Helper()
	res, err := api.SetEditorialApproval(context.Background(), connect.NewRequest(&apiv1.SetEditorialApprovalRequest{
		SessionId: sessionID, Approval: technical, Approved: approved,
	}))
	require.NoError(t, err)
	return res.Msg.UndoToken
}

func undoTestApproval(api *EditorialAPI, sessionID, token string) error {
	_, err := api.UndoEditorialApproval(context.Background(), connect.NewRequest(&apiv1.UndoEditorialApprovalRequest{
		SessionId: sessionID, UndoToken: token,
	}))
	return err
}

func technicalApproval(t *testing.T, api *EditorialAPI, sessionID string) *apiv1.EditorialApproval {
	t.Helper()
	got, err := api.loadSession(context.Background(), sessionID)
	require.NoError(t, err)
	return got.TechnicalApproval
}

// asSomeoneElse runs fn as someone@bcc.media with the given editorial role,
// then switches back to the test's own user (editor@bcc.media, admin).
func asSomeoneElse(t *testing.T, editorialJSON string, fn func()) {
	t.Helper()
	oldPerms, oldEmail := PermissionsFile, os.Getenv("DEBUG_AUTH_EMAIL")
	actAs(t, editorialJSON)
	defer func() {
		PermissionsFile = oldPerms
		t.Setenv("DEBUG_AUTH_EMAIL", oldEmail)
	}()
	fn()
}

func requireFailedPrecondition(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestEditorialUndoApprove(t *testing.T) {
	api := newTestEditorialAPI(t)
	sess := createTestSession(t, api, "VX-9", "stream")

	token := setTestApproval(t, api, sess.Id, true)
	require.NotEmpty(t, token)
	require.NoError(t, undoTestApproval(api, sess.Id, token))

	got := technicalApproval(t, api, sess.Id)
	assert.False(t, got.Approved)
	assert.Empty(t, got.By)

	// A token works once.
	requireFailedPrecondition(t, undoTestApproval(api, sess.Id, token))
}

// Undoing a removal puts back who approved it and when, not the person
// undoing it now.
func TestEditorialUndoRemoveRestoresOriginalApprover(t *testing.T) {
	api := newTestEditorialAPI(t)
	sess := createTestSession(t, api, "VX-9", "stream")
	setTestApproval(t, api, sess.Id, true)
	original := technicalApproval(t, api, sess.Id)
	require.Equal(t, "editor@bcc.media", original.By)

	asSomeoneElse(t, `{"enabled":true}`, func() {
		token := setTestApproval(t, api, sess.Id, false)
		require.False(t, technicalApproval(t, api, sess.Id).Approved)
		require.NoError(t, undoTestApproval(api, sess.Id, token))
	})

	got := technicalApproval(t, api, sess.Id)
	assert.True(t, got.Approved)
	assert.Equal(t, "editor@bcc.media", got.By)
	assert.True(t, original.At.AsTime().Equal(got.At.AsTime()))
}

// Approving an approved session keeps the first approver, and there is
// nothing to undo.
func TestEditorialReapproveKeepsFirstApprover(t *testing.T) {
	api := newTestEditorialAPI(t)
	sess := createTestSession(t, api, "VX-9", "stream")
	setTestApproval(t, api, sess.Id, true)
	first := technicalApproval(t, api, sess.Id)

	asSomeoneElse(t, `{"enabled":true}`, func() {
		assert.Empty(t, setTestApproval(t, api, sess.Id, true))
	})

	got := technicalApproval(t, api, sess.Id)
	assert.Equal(t, "editor@bcc.media", got.By)
	assert.True(t, first.At.AsTime().Equal(got.At.AsTime()))
}

func TestEditorialUndoSomeoneElsesToken(t *testing.T) {
	api := newTestEditorialAPI(t)
	sess := createTestSession(t, api, "VX-9", "stream")
	token := setTestApproval(t, api, sess.Id, true)

	asSomeoneElse(t, `{"admin":true}`, func() {
		requireFailedPrecondition(t, undoTestApproval(api, sess.Id, token))
	})
	other := createTestSession(t, api, "VX-10", "other")
	requireFailedPrecondition(t, undoTestApproval(api, other.Id, token))
	// Neither attempt used it up.
	require.NoError(t, undoTestApproval(api, sess.Id, token))
	assert.False(t, technicalApproval(t, api, sess.Id).Approved)
}

// A change made after the one being undone is never overwritten.
func TestEditorialUndoAfterChangeFails(t *testing.T) {
	api := newTestEditorialAPI(t)
	sess := createTestSession(t, api, "VX-9", "stream")
	token := setTestApproval(t, api, sess.Id, true)

	asSomeoneElse(t, `{"enabled":true}`, func() {
		setTestApproval(t, api, sess.Id, false)
		setTestApproval(t, api, sess.Id, true)
	})

	requireFailedPrecondition(t, undoTestApproval(api, sess.Id, token))
	assert.Equal(t, "someone@bcc.media", technicalApproval(t, api, sess.Id).By)
}

// A session deleted after the change is reported as gone, not as changed.
func TestEditorialUndoDeletedSession(t *testing.T) {
	api := newTestEditorialAPI(t)
	sess := createTestSession(t, api, "VX-9", "stream")
	token := setTestApproval(t, api, sess.Id, true)

	_, err := api.DeleteEditorialSession(context.Background(), connect.NewRequest(&apiv1.DeleteEditorialSessionRequest{Id: sess.Id}))
	require.NoError(t, err)
	requireNotFound(t, undoTestApproval(api, sess.Id, token))
}

func TestEditorialUndoExpired(t *testing.T) {
	api := newTestEditorialAPI(t)
	sess := createTestSession(t, api, "VX-9", "stream")
	token := setTestApproval(t, api, sess.Id, true)

	later := time.Now().Add(approvalUndoTTL + time.Minute)
	api.approvalUndos.now = func() time.Time { return later }
	requireFailedPrecondition(t, undoTestApproval(api, sess.Id, token))
	assert.True(t, technicalApproval(t, api, sess.Id).Approved)
}

func TestEditorialUndoUnknownToken(t *testing.T) {
	api := newTestEditorialAPI(t)
	sess := createTestSession(t, api, "VX-9", "stream")
	requireFailedPrecondition(t, undoTestApproval(api, sess.Id, "no-such-token"))
}
