package members

import "context"

// Person is what a contributor needs of a member: this is personal data.
type Person struct {
	// the Core API uid, a UUID
	UID         string
	DisplayName string
}

// Searcher finds people by name, best matches first, or by uid.
type Searcher interface {
	Search(ctx context.Context, query string, limit int) ([]Person, error)
	GetMany(ctx context.Context, ids []string) ([]Person, error)
}
