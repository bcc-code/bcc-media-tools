package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"context"
	"fmt"
	"sync"

	"connectrpc.com/connect"
)

// MarkersAPI is a placeholder, in-memory implementation of the markers store so
// the frontend can develop against a real contract.
//
// TODO: replace the in-memory map + demo seed with the real integration:
//   - GetMarkers should merge markers imported from the third-party timing
//     program (name-supers, bible-verse references, …) with manually-created
//     ones, keyed by VX-id.
//   - SubmitMarkers should persist edits and reconcile IMPORTED markers with
//     their source.
type MarkersAPI struct {
	mu    sync.Mutex
	store map[string][]*apiv1.Marker
}

func NewMarkersAPI() *MarkersAPI {
	return &MarkersAPI{store: map[string][]*apiv1.Marker{}}
}

func (m *MarkersAPI) GetMarkers(ctx context.Context, req *connect.Request[apiv1.GetMarkersRequest]) (*connect.Response[apiv1.GetMarkersResponse], error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return connect.NewResponse(&apiv1.GetMarkersResponse{Markers: m.store[req.Msg.VXID]}), nil
}

func (m *MarkersAPI) SubmitMarkers(ctx context.Context, req *connect.Request[apiv1.SubmitMarkersRequest]) (*connect.Response[apiv1.Void], error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.store[req.Msg.VXID] = req.Msg.Markers
	fmt.Printf("Stored %d markers for VXID %s\n", len(req.Msg.Markers), req.Msg.VXID)

	return connect.NewResponse(&apiv1.Void{}), nil
}
