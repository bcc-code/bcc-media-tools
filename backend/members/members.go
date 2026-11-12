// Package members finds people in BCC Members, through the BCC Core API.
package members

import (
	"context"
	"errors"
	"time"
)

// Person is what a contributor needs of a member: this is personal data.
type Person struct {
	// the numeric Core API personID, as Playout's SPEAKER data sends it
	PersonID    string
	DisplayName string
	// zero when unknown; only for telling who is U26, never stored or shown
	BirthDate time.Time
}

// ErrNotFound is Get's error for a personID Members doesn't know.
var ErrNotFound = errors.New("member not found")

// Searcher finds people by name, best matches first, or by personID.
type Searcher interface {
	Search(ctx context.Context, query string, limit int) ([]Person, error)
	Get(ctx context.Context, personID string) (Person, error)
}
