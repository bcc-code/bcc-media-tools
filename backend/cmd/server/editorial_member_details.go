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
	// people looked up are kept this long
	memberDetailsKeep = 24 * time.Hour
	// lookups for one answer give up after this
	memberDetailsWait = 2 * time.Second
)

// memberDetails keeps people looked up in BCC Members, for their age and
// church.
type memberDetails struct {
	members members.Searcher
	loc     *time.Location

	mu   sync.Mutex
	byID map[string]cachedPerson
}

// cachedPerson is a zero Person when Members doesn't know the ID.
type cachedPerson struct {
	members.Person
	expires time.Time
}

// newMemberDetails is nil, knowing no one, without Members.
func newMemberDetails(m members.Searcher) *memberDetails {
	if m == nil {
		return nil
	}
	loc, err := time.LoadLocation(recordingTZ)
	if err != nil {
		loc = time.UTC
	}
	return &memberDetails{members: m, loc: loc, byID: map[string]cachedPerson{}}
}

// remember keeps the people a search brought along.
func (d *memberDetails) remember(people []members.Person) {
	if d == nil {
		return
	}
	expires := time.Now().Add(memberDetailsKeep)
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, p := range people {
		d.byID[p.PersonID] = cachedPerson{Person: p, expires: expires}
	}
}

// get returns the people of ids, by personID. Those not kept are looked up
// together; when that fails or takes longer than memberDetailsWait, they're
// left out, and asked for again next time.
func (d *memberDetails) get(ctx context.Context, ids []string) map[string]members.Person {
	if d == nil || len(ids) == 0 {
		return nil
	}
	d.lookUp(ctx, ids)
	d.mu.Lock()
	defer d.mu.Unlock()
	out := map[string]members.Person{}
	for _, id := range ids {
		if p, ok := d.byID[id]; ok && p.PersonID != "" {
			out[id] = p.Person
		}
	}
	return out
}

// lookUp asks Members for the ids not kept, or kept too long.
func (d *memberDetails) lookUp(ctx context.Context, ids []string) {
	now := time.Now()
	d.mu.Lock()
	var missing []string
	for _, id := range ids {
		if p, ok := d.byID[id]; (!ok || now.After(p.expires)) && !slices.Contains(missing, id) {
			missing = append(missing, id)
		}
	}
	d.mu.Unlock()
	if len(missing) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, memberDetailsWait)
	defer cancel()
	people, err := d.members.GetMany(ctx, missing)
	if err != nil {
		log.L.Debug().Err(err).Msg("member lookup failed")
		return
	}
	expires := time.Now().Add(memberDetailsKeep)
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, id := range missing {
		d.byID[id] = cachedPerson{expires: expires}
	}
	for _, p := range people {
		d.byID[p.PersonID] = cachedPerson{Person: p, expires: expires}
	}
}

// today is the date in the meetings' time zone.
func (d *memberDetails) today() time.Time {
	return time.Now().In(d.loc)
}

// age is the years since birth by the date today; 0 when birth is unknown.
// Someone born on 29 February has their birthday on 1 March in other years.
func age(birth, today time.Time) int {
	if birth.IsZero() {
		return 0
	}
	y, m, day := today.Date()
	years := y - birth.Year()
	if m < birth.Month() || m == birth.Month() && day < birth.Day() {
		years--
	}
	return years
}

func isU26(birth, today time.Time) bool {
	return !birth.IsZero() && age(birth, today) < 26
}

// canSeeMembers: details from Members need what SearchMembers needs.
func canSeeMembers(email string) bool {
	p := PermissionsForEmail(email)
	return p.EditorialRole() >= apiv1.EditorialEditor && p.CanLookUpMembers()
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
	out := map[string]bool{}
	for id, p := range e.people.get(ctx, ids) {
		out[id] = isU26(p.BirthDate, e.people.today())
	}
	return out
}

// addMemberDetails marks the session's linked contributors U26, and with
// ageAndChurch sets those too
func (e EditorialAPI) addMemberDetails(ctx context.Context, markers []*apiv1.EditorialMarker, ageAndChurch bool) {
	var ids []string
	for _, m := range markers {
		for _, c := range m.GetContributorList() {
			if c.GetPersonId() != "" {
				ids = append(ids, c.GetPersonId())
			}
		}
	}
	people := e.people.get(ctx, ids)
	for _, m := range markers {
		for _, c := range m.GetContributorList() {
			p, ok := people[c.GetPersonId()]
			if !ok {
				continue
			}
			c.U26 = isU26(p.BirthDate, e.people.today())
			if ageAndChurch {
				c.Age = int32(age(p.BirthDate, e.people.today()))
				c.Church = p.Church
			}
		}
	}
}
