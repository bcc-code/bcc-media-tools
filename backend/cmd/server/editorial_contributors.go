package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/editorial"
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
)

// contributor is one of a marker's contributors; PersonID is empty when typed in
type contributor struct {
	Name     string
	PersonID string
}

// markerRow is a marker with its contributors. Reassign Contributors, never
// change it in place: row copies share the slice.
type markerRow struct {
	editorial.Marker
	Contributors []contributor
}

func (m markerRow) equal(o markerRow) bool {
	return m.Marker == o.Marker && slices.Equal(m.Contributors, o.Contributors)
}

// contributorString is the names joined with ", ", as the API and the Excel
// export show them.
func (m markerRow) contributorString() string {
	return strings.Join(contributorNames(m.Contributors), ", ")
}

// listMarkers returns a session's markers in order, with their contributors.
func listMarkers(ctx context.Context, q *editorial.Queries, sessionID string) ([]markerRow, error) {
	markers, err := q.ListMarkersForSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("editorial: list markers: %w", err)
	}
	contributors, err := q.ListContributorsForSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("editorial: list contributors: %w", err)
	}
	byMarker := map[string][]contributor{}
	for _, c := range contributors {
		byMarker[c.MarkerID] = append(byMarker[c.MarkerID], contributor{Name: c.Name, PersonID: c.PersonID.String})
	}
	rows := make([]markerRow, len(markers))
	for i, m := range markers {
		rows[i] = markerRow{Marker: m, Contributors: byMarker[m.ID]}
	}
	return rows, nil
}

// writeContributors stores a marker's contributors; it has none stored yet.
func writeContributors(ctx context.Context, q *editorial.Queries, markerID string, list []contributor) error {
	for i, c := range list {
		if err := q.InsertMarkerContributor(ctx, editorial.InsertMarkerContributorParams{
			MarkerID: markerID,
			Position: int64(i),
			Name:     c.Name,
			PersonID: sql.NullString{String: c.PersonID, Valid: c.PersonID != ""},
		}); err != nil {
			return fmt.Errorf("editorial: insert contributor: %w", err)
		}
	}
	return nil
}

// contributorsFromRequest trims the names and drops blank ones.
func contributorsFromRequest(list []*apiv1.EditorialContributor) []contributor {
	var out []contributor
	for _, c := range list {
		if name := strings.TrimSpace(c.GetName()); name != "" {
			out = append(out, contributor{Name: name, PersonID: strings.TrimSpace(c.GetPersonId())})
		}
	}
	return out
}

func contributorNames(list []contributor) []string {
	names := make([]string, len(list))
	for i, c := range list {
		names[i] = c.Name
	}
	return names
}

func contributorsToProto(list []contributor) []*apiv1.EditorialContributor {
	out := make([]*apiv1.EditorialContributor, len(list))
	for i, c := range list {
		out[i] = &apiv1.EditorialContributor{Name: c.Name, PersonId: c.PersonID}
	}
	return out
}
