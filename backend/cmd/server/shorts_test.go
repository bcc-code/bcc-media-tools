package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"testing"
)

func TestValidateShortRange(t *testing.T) {
	tests := []struct {
		name    string
		msg     *apiv1.SubmitShortRequest
		wantErr bool
	}{
		{"valid range", &apiv1.SubmitShortRequest{VXID: "VX-1", InSeconds: 10, OutSeconds: 40}, false},
		{"starts at zero", &apiv1.SubmitShortRequest{VXID: "VX-1", InSeconds: 0, OutSeconds: 1}, false},
		{"missing VXID", &apiv1.SubmitShortRequest{InSeconds: 10, OutSeconds: 40}, true},
		{"blank VXID", &apiv1.SubmitShortRequest{VXID: "   ", InSeconds: 10, OutSeconds: 40}, true},
		{"negative start", &apiv1.SubmitShortRequest{VXID: "VX-1", InSeconds: -1, OutSeconds: 40}, true},
		{"zero-length range", &apiv1.SubmitShortRequest{VXID: "VX-1", InSeconds: 10, OutSeconds: 10}, true},
		{"end before start", &apiv1.SubmitShortRequest{VXID: "VX-1", InSeconds: 40, OutSeconds: 10}, true},
		{"unset range", &apiv1.SubmitShortRequest{VXID: "VX-1"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateShortRange(tt.msg)
			if tt.wantErr && err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
