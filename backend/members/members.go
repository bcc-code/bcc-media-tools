package members

import (
	"context"
	"time"
)

// Person is what a contributor needs of a member: this is personal data.
type Person struct {
	// the Core API uid, a UUID
	UID         string
	DisplayName string
	// zero when unknown; for working out the age, never stored or shown
	BirthDate time.Time
	// local church; empty when unknown
	Church string
}

// Age is the years since birth by today's date, in today's location; 0 when
// the birth date is unknown. Someone born on 29 February has their birthday
// on 1 March in other years.
func (p Person) Age(today time.Time) int {
	if p.BirthDate.IsZero() {
		return 0
	}
	y, m, day := today.Date()
	years := y - p.BirthDate.Year()
	if m < p.BirthDate.Month() || m == p.BirthDate.Month() && day < p.BirthDate.Day() {
		years--
	}
	return years
}

// IsU26 reports whether p is known to be under 26 today.
func (p Person) IsU26(today time.Time) bool {
	return !p.BirthDate.IsZero() && p.Age(today) < 26
}

// Searcher finds people by name, best matches first, or by uid.
type Searcher interface {
	Search(ctx context.Context, query string, limit int) ([]Person, error)
	GetMany(ctx context.Context, ids []string) ([]Person, error)
}
