package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/editorial"
	"bcc-media-tools/members"
	"bcc-media-tools/playout"
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/bcc-code/mediabank-bridge/log"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func date(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse(time.DateOnly, s)
	require.NoError(t, err)
	return d
}

// yearsAgo is a birth date that many years before today.
func yearsAgo(years int) time.Time {
	y, m, d := time.Now().AddDate(-years, 0, 0).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// Made-up people: 90001 is 20, 90002 is 40, 90003 has no birth date.
func agesFake() *fakeMembers {
	return &fakeMembers{byID: map[string]members.Person{
		"90001": {PersonID: "90001", DisplayName: "Ola Nordmann", BirthDate: yearsAgo(20)},
		"90002": {PersonID: "90002", DisplayName: "Kari Nordmann", BirthDate: yearsAgo(40)},
		"90003": {PersonID: "90003", DisplayName: "Per Hansen"},
	}}
}

func TestIsU26(t *testing.T) {
	today := date(t, "2026-11-12")
	assert.True(t, isU26(date(t, "2000-11-13"), today), "26 tomorrow")
	assert.False(t, isU26(date(t, "2000-11-12"), today), "26 today")
	assert.False(t, isU26(date(t, "1990-01-01"), today))
	assert.False(t, isU26(time.Time{}, today), "unknown")

	leap := date(t, "2000-02-29")
	assert.True(t, isU26(leap, date(t, "2026-02-28")))
	assert.False(t, isU26(leap, date(t, "2026-03-01")), "26 on 1 March")

	// The date in Oslo counts, not the time of day.
	oslo, err := time.LoadLocation(recordingTZ)
	require.NoError(t, err)
	assert.True(t, isU26(date(t, "2000-11-13"), time.Date(2026, 11, 12, 23, 59, 0, 0, oslo)))
}

func TestMemberAgesLooksEachUpOnce(t *testing.T) {
	log.ConfigureGlobalLogger(zerolog.Disabled)
	fake := agesFake()
	ages := newMemberAges(fake)

	ids := []string{"90001", "90002", "90003", "90009"}
	assert.Equal(t, map[string]bool{"90001": true}, ages.u26(context.Background(), ids))
	assert.Equal(t, map[string]bool{"90001": true}, ages.u26(context.Background(), ids))
	for _, id := range ids {
		assert.Equal(t, 1, fake.getCount(id), "%s looked up once, unknown ones too", id)
	}

	// A search's birth dates need no lookup.
	ages.remember([]members.Person{{PersonID: "90004", BirthDate: yearsAgo(18)}})
	assert.True(t, ages.u26(context.Background(), []string{"90004"})["90004"])
	assert.Zero(t, fake.getCount("90004"))

	assert.Nil(t, newMemberAges(nil).u26(context.Background(), ids), "no Members, no one marked")
}

// A slow Members gives no marks rather than a slow page, and is asked again
// next time.
func TestMemberAgesDoesNotWaitLong(t *testing.T) {
	log.ConfigureGlobalLogger(zerolog.Disabled)
	wait := agesWait
	agesWait = 20 * time.Millisecond
	t.Cleanup(func() { agesWait = wait })
	fake := agesFake()
	fake.delay = time.Second
	ages := newMemberAges(fake)

	start := time.Now()
	assert.Empty(t, ages.u26(context.Background(), []string{"90001", "90002"}))
	assert.Less(t, time.Since(start), 500*time.Millisecond)

	fake.delay = 0
	assert.True(t, ages.u26(context.Background(), []string{"90001"})["90001"])
	assert.Equal(t, 2, fake.getCount("90001"))
}

func TestGetEditorialSessionMarksU26(t *testing.T) {
	log.ConfigureGlobalLogger(zerolog.Disabled)
	api := newTestEditorialAPI(t)
	api.ages = newMemberAges(agesFake())
	sess := createTestSession(t, api, "VX-1", "Søndag")
	saveTestSession(t, api, sess.Id, "Søndag", []*apiv1.EditorialMarker{{
		Name: "Tale", StartMs: 0, EndMs: 60_000,
		ContributorList: []*apiv1.EditorialContributor{
			{Name: "Ola Nordmann", PersonId: "90001"},
			{Name: "Kari Nordmann", PersonId: "90002"},
			{Name: "Per Hansen", PersonId: "90003"},
			{Name: "Ola Nordmann"},
			// a client can't mark anyone
			{Name: "Ola Olsen", PersonId: "90009", U26: true},
		},
	}})

	res, err := api.GetEditorialSession(context.Background(), connect.NewRequest(&apiv1.GetEditorialSessionRequest{Id: sess.Id}))
	require.NoError(t, err)
	var got []bool
	for _, c := range res.Msg.Markers[0].ContributorList {
		got = append(got, c.U26)
	}
	assert.Equal(t, []bool{true, false, false, false, false}, got, "only the linked 20-year-old")
}

func TestSearchMembersMarksU26(t *testing.T) {
	log.ConfigureGlobalLogger(zerolog.Disabled)
	noSpeakerWaits(t)
	api := newTestEditorialAPI(t)
	api.speakers = newFrequentSpeakers(&playoutEvents{
		events:    []playout.Event{{ID: "1", Status: "archived"}},
		manifests: map[string][]playout.ManifestEntry{"1": {speakerAt(t, "Ola Nordmann", `"90001"`, "2026-01-10T10:00:00Z")}},
	})
	require.NoError(t, api.speakers.refresh(context.Background()))
	fake := agesFake()
	fake.people = []members.Person{
		{PersonID: "90005", DisplayName: "Ola Olsen", BirthDate: yearsAgo(19)},
		{PersonID: "90006", DisplayName: "Ola Berg", BirthDate: yearsAgo(30)},
	}
	api.members = fake
	api.ages = newMemberAges(fake)

	got, err := searchMembers(t, api, "ola", 0)
	require.NoError(t, err)
	marks := map[string]bool{}
	for _, m := range got {
		marks[m.PersonId] = m.U26
	}
	assert.Equal(t, map[string]bool{"90001": true, "90005": true, "90006": false}, marks)
	assert.Equal(t, 1, fake.getCount("90001"), "a frequent speaker is looked up")
	assert.Zero(t, fake.getCount("90005"), "a search result brings its birth date")
}

func TestEditorialWorkbookU26Column(t *testing.T) {
	sess := editorial.Session{Vxid: "VX-1", Kind: editorial.KindRecording}
	var m editorial.Marker
	m.EndMs = 60_000
	setContributors(&m, []contributor{{Name: "Ola Nordmann", PersonID: "90001"}, {Name: "Kari Nordmann", PersonID: "90002"}, {Name: "Per Hansen"}})
	data, err := editorialWorkbook(sess, []editorial.Marker{m}, map[string]bool{"90001": true}, exportLabelsByLanguage["en"])
	require.NoError(t, err)
	f, sheet := openWorkbook(t, data)
	rows, err := f.GetRows(sheet)
	require.NoError(t, err)
	assert.Equal(t, []string{"Contributors", "U26"}, rows[0][4:6])
	assert.Equal(t, []string{"Ola Nordmann, Kari Nordmann, Per Hansen", "Ola Nordmann"}, rows[1][4:6])
}
