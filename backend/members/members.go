// Package members finds people in BCC Members, through the BCC Core API.
package members

import "context"

// Person is what a contributor needs of a member: this is personal data.
type Person struct {
	// the numeric Core API personID, as Playout's SPEAKER data sends it
	PersonID    string
	DisplayName string
}

// Searcher finds people by name, best matches first.
type Searcher interface {
	Search(ctx context.Context, query string, limit int) ([]Person, error)
}
