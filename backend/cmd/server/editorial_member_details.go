package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/editorial"
	"bcc-media-tools/members"
	"context"
	"sync"
	"time"

	"github.com/bcc-code/mediabank-bridge/log"
)

// meetingsTZ is the meetings' time zone, where birthdays start.
var meetingsTZ = sync.OnceValue(func() *time.Location {
	loc, err := time.LoadLocation(recordingTZ)
	if err != nil {
		return time.UTC
	}
	return loc
})

func today() time.Time {
	return time.Now().In(meetingsTZ())
}

// lookUpMembers gets people from e.people, leaving out those it can't.
func (e EditorialAPI) lookUpMembers(ctx context.Context, uids []string) map[string]members.Person {
	people, err := e.people.Get(ctx, uids)
	if err != nil {
		log.L.Debug().Err(err).Msg("member lookup failed")
	}
	return people
}

// canSeeMembers: details from Members need what SearchMembers needs.
func canSeeMembers(email string) bool {
	p := PermissionsForEmail(email)
	return p.EditorialRole() >= apiv1.EditorialEditor && p.CanLookUpMembers()
}

// u26Contributors tells which of the markers' linked contributors are U26.
func (e EditorialAPI) u26Contributors(ctx context.Context, markers []editorial.MarkerRow) map[string]bool {
	var ids []string
	for _, m := range markers {
		for _, c := range m.Contributors {
			if c.MemberUID != "" {
				ids = append(ids, c.MemberUID)
			}
		}
	}
	out := map[string]bool{}
	for id, p := range e.lookUpMembers(ctx, ids) {
		out[id] = p.IsU26(today())
	}
	return out
}

// addMemberDetails marks the session's linked contributors U26, and with
// ageAndChurch sets those too
func (e EditorialAPI) addMemberDetails(ctx context.Context, markers []*apiv1.EditorialMarker, ageAndChurch bool) {
	var ids []string
	for _, m := range markers {
		for _, c := range m.GetContributorList() {
			if c.GetMemberUid() != "" {
				ids = append(ids, c.GetMemberUid())
			}
		}
	}
	people := e.lookUpMembers(ctx, ids)
	for _, m := range markers {
		for _, c := range m.GetContributorList() {
			p, ok := people[c.GetMemberUid()]
			if !ok {
				continue
			}
			c.U26 = p.IsU26(today())
			if ageAndChurch {
				c.Age = int32(p.Age(today()))
				c.Church = p.Church
			}
		}
	}
}
