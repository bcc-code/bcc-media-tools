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

// Age is the years since birth on date, in date's location; 0 when
// the birth date is unknown. Someone born on 29 February has their birthday
// on 1 March in other years.
func (p Person) Age(date time.Time) int {
	if p.BirthDate.IsZero() {
		return 0
	}
	y, m, day := date.Date()
	years := y - p.BirthDate.Year()
	if m < p.BirthDate.Month() || m == p.BirthDate.Month() && day < p.BirthDate.Day() {
		years--
	}
	return years
}

// IsU26 reports whether p is known to be under 26 on date.
func (p Person) IsU26(date time.Time) bool {
	return !p.BirthDate.IsZero() && p.Age(date) < 26
}

// Searcher finds people by name, best matches first, or by uid.
type Searcher interface {
	Search(ctx context.Context, query string, limit int) ([]Person, error)
	GetMany(ctx context.Context, ids []string) ([]Person, error)
}
