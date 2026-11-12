package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/members"
	"context"
	"slices"
	"sync"
	"time"

	"github.com/bcc-code/mediabank-bridge/log"
)

// Vars for tests.
var (
	// birth dates are kept this long
	agesKeep = 24 * time.Hour
	// lookups for one answer give up after this
	agesWait = 2 * time.Second
)

// memberAges tells who is under 26 today, from birth dates in BCC Members.
// Birth dates stay in memory: never stored, never sent to the browser.
type memberAges struct {
	members members.Searcher
	loc     *time.Location

	mu   sync.Mutex
	born map[string]birthDate
}

type birthDate struct {
	// zero when unknown
	date    time.Time
	expires time.Time
}

// newMemberAges is nil, marking no one, without Members.
func newMemberAges(m members.Searcher) *memberAges {
	if m == nil {
		return nil
	}
	loc, err := time.LoadLocation(recordingTZ)
	if err != nil {
		loc = time.UTC
	}
	return &memberAges{members: m, loc: loc, born: map[string]birthDate{}}
}

// remember keeps the birth dates a search brought along.
func (a *memberAges) remember(people []members.Person) {
	if a == nil {
		return
	}
	expires := time.Now().Add(agesKeep)
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, p := range people {
		a.born[p.PersonID] = birthDate{date: p.BirthDate, expires: expires}
	}
}

// u26 reports who of ids is under 26 today. Birth dates not kept are looked
// up together; when that fails or takes longer than agesWait, those count as
// not, and are asked for again next time.
func (a *memberAges) u26(ctx context.Context, ids []string) map[string]bool {
	if a == nil || len(ids) == 0 {
		return nil
	}
	a.lookUp(ctx, ids)
	today := time.Now().In(a.loc)
	a.mu.Lock()
	defer a.mu.Unlock()
	out := map[string]bool{}
	for _, id := range ids {
		if isU26(a.born[id].date, today) {
			out[id] = true
		}
	}
	return out
}

// lookUp asks Members for the birth dates of ids not kept, or kept too long.
func (a *memberAges) lookUp(ctx context.Context, ids []string) {
	now := time.Now()
	a.mu.Lock()
	var missing []string
	for _, id := range ids {
		if b, ok := a.born[id]; (!ok || now.After(b.expires)) && !slices.Contains(missing, id) {
			missing = append(missing, id)
		}
	}
	a.mu.Unlock()
	if len(missing) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, agesWait)
	defer cancel()
	people, err := a.members.GetMany(ctx, missing)
	if err != nil {
		log.L.Debug().Err(err).Msg("member birth date lookup failed")
		return
	}
	expires := time.Now().Add(agesKeep)
	a.mu.Lock()
	defer a.mu.Unlock()
	// those Members doesn't know have no birth date
	for _, id := range missing {
		a.born[id] = birthDate{expires: expires}
	}
	for _, p := range people {
		a.born[p.PersonID] = birthDate{date: p.BirthDate, expires: expires}
	}
}

// isU26: the 26th birthday is still to come, by the date today. Someone born
// on 29 February turns 26 on 1 March in other years.
func isU26(birth, today time.Time) bool {
	if birth.IsZero() {
		return false
	}
	y, m, d := today.Date()
	return birth.AddDate(26, 0, 0).After(time.Date(y, m, d, 0, 0, 0, 0, time.UTC))
}

// u26Contributors tells which of the markers' linked contributors are U26.
func (e EditorialAPI) u26Contributors(ctx context.Context, markers []markerRow) map[string]bool {
	var ids []string
	for _, m := range markers {
		for _, c := range m.Contributors {
			if c.PersonID != "" {
				ids = append(ids, c.PersonID)
			}
		}
	}
	return e.ages.u26(ctx, ids)
}

// markU26 sets u26 on the session's linked contributors.
func (e EditorialAPI) markU26(ctx context.Context, markers []*apiv1.EditorialMarker) {
	var ids []string
	for _, m := range markers {
		for _, c := range m.GetContributorList() {
			if c.GetPersonId() != "" {
				ids = append(ids, c.GetPersonId())
			}
		}
	}
	u26 := e.ages.u26(ctx, ids)
	for _, m := range markers {
		for _, c := range m.GetContributorList() {
			c.U26 = u26[c.GetPersonId()]
		}
	}
}
