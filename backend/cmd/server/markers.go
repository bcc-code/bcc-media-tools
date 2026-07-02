package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"context"
	"fmt"
	"sync"

	"connectrpc.com/connect"
	"github.com/bcc-code/mediabank-bridge/log"
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

	bible *BibleClient
}

func NewMarkersAPI(bible *BibleClient) *MarkersAPI {
	return &MarkersAPI{
		store: map[string][]*apiv1.Marker{},
		bible: bible,
	}
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

const defaultEntitySearchLimit = 10
const maxEntitySearchLimit = 25

// SearchEntities powers marker-label autocomplete. Today only bible-verse
// lookups are wired up (against the BCC bible server); song and people
// registries can be added as further cases without changing the contract.
func (m *MarkersAPI) SearchEntities(ctx context.Context, req *connect.Request[apiv1.SearchEntitiesRequest]) (*connect.Response[apiv1.SearchEntitiesResponse], error) {
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = defaultEntitySearchLimit
	}
	if limit > maxEntitySearchLimit {
		limit = maxEntitySearchLimit
	}

	var entities []*apiv1.Entity
	switch req.Msg.Type {
	case apiv1.MarkerType_MARKER_TYPE_BIBLE_VERSE:
		found, err := m.bible.Search(ctx, req.Msg.Query, limit)
		if err != nil {
			// Autocomplete is best-effort: log and return no suggestions rather
			// than failing the request (the user can still type free text).
			log.L.Warn().Err(err).Msg("bible search failed")
			break
		}
		entities = found
	}

	return connect.NewResponse(&apiv1.SearchEntitiesResponse{Entities: entities}), nil
}

// ResolveReferences bulk-resolves free-text marker labels to canonical entities
// for the "resolve references" action. Unresolvable entries come back with
// resolved=false; the whole request never fails on a per-entry lookup error.
func (m *MarkersAPI) ResolveReferences(ctx context.Context, req *connect.Request[apiv1.ResolveReferencesRequest]) (*connect.Response[apiv1.ResolveReferencesResponse], error) {
	results := make([]*apiv1.ResolvedReference, 0, len(req.Msg.Queries))
	for _, q := range req.Msg.Queries {
		res := &apiv1.ResolvedReference{RefId: q.RefId}
		switch q.Type {
		case apiv1.MarkerType_MARKER_TYPE_BIBLE_VERSE:
			entity, err := m.bible.Resolve(ctx, q.Text)
			if err != nil {
				log.L.Warn().Err(err).Msg("bible resolve failed")
				break
			}
			if entity != nil {
				res.Resolved = true
				res.Entity = entity
			}
		}
		results = append(results, res)
	}

	return connect.NewResponse(&apiv1.ResolveReferencesResponse{Results: results}), nil
}
