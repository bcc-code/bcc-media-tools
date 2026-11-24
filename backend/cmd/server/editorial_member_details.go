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

// recordingDate is the day sess's recording started, in the meetings' time
// zone: ages and U26 are as of then, not today. First known of: the live
// clock start, a hand-entered window, the asset's ingest date in Vidispine,
// and the day the session was made.
func (e EditorialAPI) recordingDate(sess editorial.Session) time.Time {
	at := func(ms int64) time.Time { return time.UnixMilli(ms).In(meetingsTZ()) }
	if sess.ClockStartMs != 0 {
		return at(sess.ClockStartMs)
	}
	if sess.RecordingStartMs != 0 {
		return at(sess.RecordingStartMs)
	}
	if d, ok := e.assetDate(sess.Vxid); ok {
		return d
	}
	return at(sess.CreatedAt)
}

// assetDate is the day a Vidispine item was recorded, remembered as it never
// changes.
func (e EditorialAPI) assetDate(vxID string) (time.Time, bool) {
	if vxID == "" || e.vidispine == nil || e.assetDates == nil {
		return time.Time{}, false
	}
	if d, ok := e.assetDates.Load(vxID); ok {
		return d.(time.Time), true
	}
	meta, err := fetchRecordingMetadata(e.vidispine, vxID)
	if err != nil {
		log.L.Debug().Err(err).Str("vxid", vxID).Msg("recording date lookup failed")
		return time.Time{}, false
	}
	y, m, day, err := recordingDateFromMetadata(meta, meetingsTZ())
	if err != nil {
		log.L.Debug().Err(err).Str("vxid", vxID).Msg("no recording date")
		return time.Time{}, false
	}
	d := time.Date(y, m, day, 0, 0, 0, 0, meetingsTZ())
	e.assetDates.Store(vxID, d)
	return d, true
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

// u26Contributors tells which of the markers' linked contributors were U26 on
// date.
func (e EditorialAPI) u26Contributors(ctx context.Context, markers []editorial.MarkerRow, date time.Time) map[string]bool {
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
		out[id] = p.IsU26(date)
	}
	return out
}

// addMemberDetails marks the session's linked contributors U26 on date, and
// with ageAndChurch sets their age on date and church too
func (e EditorialAPI) addMemberDetails(ctx context.Context, markers []*apiv1.EditorialMarker, date time.Time, ageAndChurch bool) {
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
			c.U26 = p.IsU26(date)
			if ageAndChurch {
				c.Age = int32(p.Age(date))
				c.Church = p.Church
			}
		}
	}
}
