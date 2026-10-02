package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"testing"

	"connectrpc.com/connect"
)

func TestTranscriptionAccess(t *testing.T) {
	volunteer := &apiv1.Permissions{
		Transcription: &apiv1.TranscriptionPermission{Mediabanken: true},
	}
	transcriptionAdmin := &apiv1.Permissions{
		Transcription: &apiv1.TranscriptionPermission{Admin: true},
	}
	globalAdmin := &apiv1.Permissions{
		Admin:         true,
		Transcription: &apiv1.TranscriptionPermission{Mediabanken: true},
	}

	tests := []struct {
		name   string
		email  string
		perms  *apiv1.Permissions
		shared bool
		want   connect.Code
	}{
		{"no email", "", volunteer, true, connect.CodeUnauthenticated},
		{"no permissions at all", "a@b.no", &apiv1.Permissions{}, true, connect.CodePermissionDenied},
		{"tool permission but none granted", "a@b.no", &apiv1.Permissions{
			Transcription: &apiv1.TranscriptionPermission{},
		}, true, connect.CodePermissionDenied},
		{"volunteer on a shared asset", "a@b.no", volunteer, true, 0},
		{"volunteer on an asset that is not shared", "a@b.no", volunteer, false, connect.CodePermissionDenied},
		{"transcription admin on any asset", "a@b.no", transcriptionAdmin, false, 0},
		{"global admin on any asset", "a@b.no", globalAdmin, false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := transcriptionAccess(tt.email, tt.perms, func() bool { return tt.shared })

			if tt.want == 0 {
				if err != nil {
					t.Fatalf("want access, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("want %v, got access", tt.want)
			}
			if got := connect.CodeOf(err); got != tt.want {
				t.Fatalf("want %v, got %v (%v)", tt.want, got, err)
			}
		})
	}
}

// The ACL lookup is a Cantemo round trip, so it must not happen for callers who
// are allowed through regardless of it.
func TestTranscriptionAccessSkipsACLLookupForAdmins(t *testing.T) {
	for _, perms := range []*apiv1.Permissions{
		{Admin: true, Transcription: &apiv1.TranscriptionPermission{Mediabanken: true}},
		{Transcription: &apiv1.TranscriptionPermission{Admin: true}},
	} {
		called := false
		err := transcriptionAccess("a@b.no", perms, func() bool {
			called = true
			return false
		})

		if err != nil {
			t.Fatalf("want access, got %v", err)
		}
		if called {
			t.Fatal("ACL was looked up for an admin")
		}
	}
}

// A caller without the tool permission must be rejected before any lookup.
func TestTranscriptionAccessRejectsBeforeACLLookup(t *testing.T) {
	called := false
	err := transcriptionAccess("a@b.no", &apiv1.Permissions{}, func() bool {
		called = true
		return true
	})

	if err == nil {
		t.Fatal("want rejection, got access")
	}
	if called {
		t.Fatal("ACL was looked up for a caller without the transcription permission")
	}
}
