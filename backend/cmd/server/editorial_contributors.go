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
		byMarker[c.MarkerID] = append(byMarker[c.MarkerID], contributorFromRow(c))
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

func contributorFromRow(c editorial.MarkerContributor) contributor {
	return contributor{Name: c.Name, PersonID: c.PersonID.String}
}

// contributorsFromRequest uses the list unless the string names others; then
// the string wins and keeps IDs already linked in the list or stored row.
func contributorsFromRequest(list []*apiv1.EditorialContributor, legacy string, stored []contributor) []contributor {
	listed := make([]contributor, 0, len(list))
	for _, c := range list {
		name := strings.TrimSpace(c.GetName())
		if name == "" {
			continue
		}
		listed = append(listed, contributor{Name: name, PersonID: strings.TrimSpace(c.GetPersonId())})
	}
	typed := splitContributors(legacy)
	if len(listed) > 0 && (len(typed) == 0 || slices.Equal(contributorNames(listed), contributorNames(typed))) {
		return listed
	}
	ids := map[string]string{}
	for _, c := range slices.Concat(stored, listed) {
		if c.PersonID != "" {
			ids[c.Name] = c.PersonID
		}
	}
	for i := range typed {
		typed[i].PersonID = ids[typed[i].Name]
	}
	return typed
}

func contributorNames(list []contributor) []string {
	names := make([]string, len(list))
	for i, c := range list {
		names[i] = c.Name
	}
	return names
}

func splitContributors(s string) []contributor {
	var out []contributor
	for _, name := range strings.Split(s, ",") {
		if name = strings.TrimSpace(name); name != "" {
			out = append(out, contributor{Name: name})
		}
	}
	return out
}

func contributorsToProto(list []contributor) []*apiv1.EditorialContributor {
	out := make([]*apiv1.EditorialContributor, len(list))
	for i, c := range list {
		out[i] = &apiv1.EditorialContributor{Name: c.Name, PersonId: c.PersonID}
	}
	return out
}
