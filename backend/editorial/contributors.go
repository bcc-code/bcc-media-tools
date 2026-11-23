package editorial

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"unicode"
)

// Contributor is one of a marker's contributors; PersonID is empty when typed in.
type Contributor struct {
	Name     string
	PersonID string
}

// MarkerRow is a marker with its contributors. Reassign Contributors, never
// change it in place: row copies share the slice.
type MarkerRow struct {
	Marker
	Contributors []Contributor
}

func (m MarkerRow) Equal(o MarkerRow) bool {
	return m.Marker == o.Marker && slices.Equal(m.Contributors, o.Contributors)
}

// ContributorString is the names joined with ", ", as the Excel export shows
// them.
func (m MarkerRow) ContributorString() string {
	return strings.Join(ContributorNames(m.Contributors), ", ")
}

// ListMarkerRows returns a session's markers in order, with their contributors.
func (q *Queries) ListMarkerRows(ctx context.Context, sessionID string) ([]MarkerRow, error) {
	markers, err := q.ListMarkersForSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("editorial: list markers: %w", err)
	}
	contributors, err := q.ListContributorsForSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("editorial: list contributors: %w", err)
	}
	byMarker := map[string][]Contributor{}
	for _, c := range contributors {
		byMarker[c.MarkerID] = append(byMarker[c.MarkerID], Contributor{Name: c.Name, PersonID: c.PersonID.String})
	}
	rows := make([]MarkerRow, len(markers))
	for i, m := range markers {
		rows[i] = MarkerRow{Marker: m, Contributors: byMarker[m.ID]}
	}
	return rows, nil
}

// InsertContributors stores a marker's contributors; it has none stored yet.
func (q *Queries) InsertContributors(ctx context.Context, markerID string, list []Contributor) error {
	for i, c := range list {
		if err := q.InsertMarkerContributor(ctx, InsertMarkerContributorParams{
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

// CleanContributors trims the names and person IDs and drops blank names.
func CleanContributors(list []Contributor) []Contributor {
	var out []Contributor
	for _, c := range list {
		if name := strings.TrimSpace(c.Name); name != "" {
			out = append(out, Contributor{Name: name, PersonID: strings.TrimSpace(c.PersonID)})
		}
	}
	return out
}

// SearchLinkedContributors returns the contributors picked from Members whose
// names match query, most used first. Each person comes once per name they
// were saved with.
func (q *Queries) SearchLinkedContributors(ctx context.Context, query string) ([]Contributor, error) {
	linked, err := q.ListLinkedContributors(ctx)
	if err != nil {
		return nil, fmt.Errorf("editorial: list linked contributors: %w", err)
	}
	var out []Contributor
	for _, c := range linked {
		if matchesWords(c.Name, query) {
			out = append(out, Contributor{Name: c.Name, PersonID: c.PersonID.String})
		}
	}
	return out, nil
}

// matchesWords reports whether each word of query starts a word of name,
// ignoring case: "ola nor" finds "Ola Nordmann-Berg".
func matchesWords(name, query string) bool {
	words := nameWords(strings.ToLower(name))
	return !slices.ContainsFunc(nameWords(strings.ToLower(query)), func(typed string) bool {
		return !slices.ContainsFunc(words, func(w string) bool { return strings.HasPrefix(w, typed) })
	})
}

// nameWords splits at spaces, hyphens and dots ("kari nordmann-berg").
func nameWords(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || r == '-' || r == '.'
	})
}

func ContributorNames(list []Contributor) []string {
	names := make([]string, len(list))
	for i, c := range list {
		names[i] = c.Name
	}
	return names
}
