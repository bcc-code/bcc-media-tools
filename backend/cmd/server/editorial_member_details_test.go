package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/editorial"
	"bcc-media-tools/members"
	"context"
	"fmt"
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

// Made-up people: 90001 is 20, 90002 is 40, 90003 has no birth date or
// church.
func detailsFake() *fakeMembers {
	return &fakeMembers{byID: map[string]members.Person{
		"90001": {PersonID: "90001", DisplayName: "Ola Nordmann", BirthDate: yearsAgo(20), Church: "Church One"},
		"90002": {PersonID: "90002", DisplayName: "Kari Nordmann", BirthDate: yearsAgo(40), Church: "Church Two"},
		"90003": {PersonID: "90003", DisplayName: "Per Hansen"},
	}}
}

func TestAge(t *testing.T) {
	today := date(t, "2026-11-12")
	assert.Equal(t, 25, age(date(t, "2000-11-13"), today), "26 tomorrow")
	assert.Equal(t, 26, age(date(t, "2000-11-12"), today), "26 today")
	assert.Equal(t, 36, age(date(t, "1990-01-01"), today))
	assert.Zero(t, age(time.Time{}, today), "unknown")

	leap := date(t, "2000-02-29")
	assert.Equal(t, 25, age(leap, date(t, "2026-02-28")))
	assert.Equal(t, 26, age(leap, date(t, "2026-03-01")), "26 on 1 March")

	// The date in Oslo counts, not the time of day.
	oslo, err := time.LoadLocation(recordingTZ)
	require.NoError(t, err)
	assert.Equal(t, 25, age(date(t, "2000-11-13"), time.Date(2026, 11, 12, 23, 59, 0, 0, oslo)))

	assert.True(t, isU26(date(t, "2000-11-13"), today))
	assert.False(t, isU26(date(t, "2000-11-12"), today))
	assert.False(t, isU26(time.Time{}, today), "unknown")
}

func TestMemberDetailsLooksEachUpOnce(t *testing.T) {
	log.ConfigureGlobalLogger(zerolog.Disabled)
	fake := detailsFake()
	people := newMemberDetails(fake)

	ids := []string{"90001", "90002", "90003", "90009"}
	for range 2 {
		got := people.get(context.Background(), ids)
		assert.Equal(t, []string{"Church One", "Church Two", ""}, []string{got["90001"].Church, got["90002"].Church, got["90003"].Church})
		assert.NotContains(t, got, "90009", "unknown to Members")
	}
	for _, id := range ids {
		assert.Equal(t, 1, fake.getCount(id), "%s looked up once, unknown ones too", id)
	}

	// A search's people need no lookup.
	people.remember([]members.Person{{PersonID: "90004", Church: "Church Four"}})
	assert.Equal(t, "Church Four", people.get(context.Background(), []string{"90004"})["90004"].Church)
	assert.Zero(t, fake.getCount("90004"))

	assert.Nil(t, newMemberDetails(nil).get(context.Background(), ids), "no Members, no one known")
}

// A slow Members gives no details rather than a slow page, and is asked again
// next time.
func TestMemberDetailsDoesNotWaitLong(t *testing.T) {
	log.ConfigureGlobalLogger(zerolog.Disabled)
	wait := memberDetailsWait
	memberDetailsWait = 20 * time.Millisecond
	t.Cleanup(func() { memberDetailsWait = wait })
	fake := detailsFake()
	fake.delay = time.Second
	people := newMemberDetails(fake)

	start := time.Now()
	assert.Empty(t, people.get(context.Background(), []string{"90001", "90002"}))
	assert.Less(t, time.Since(start), 500*time.Millisecond)

	fake.delay = 0
	assert.Contains(t, people.get(context.Background(), []string{"90001"}), "90001")
	assert.Equal(t, 2, fake.getCount("90001"))
}

// U26 for anyone; age and church only for those who may look up Members.
func TestGetEditorialSessionAddsDetails(t *testing.T) {
	log.ConfigureGlobalLogger(zerolog.Disabled)
	api := newTestEditorialAPI(t)
	api.people = newMemberDetails(detailsFake())
	sess := createTestSession(t, api, "VX-1", "Søndag")
	saveTestSession(t, api, sess.Id, "Søndag", []*apiv1.EditorialMarker{{
		Name: "Tale", StartMs: 0, EndMs: 60_000,
		ContributorList: []*apiv1.EditorialContributor{
			{Name: "Ola Nordmann", PersonId: "90001"},
			{Name: "Kari Nordmann", PersonId: "90002"},
			{Name: "Per Hansen", PersonId: "90003"},
			{Name: "Ola Nordmann"},
			// a client can't set them
			{Name: "Ola Olsen", PersonId: "90009", Age: 18, Church: "Made Up", U26: true},
		},
	}})
	details := func() []string {
		t.Helper()
		res, err := api.GetEditorialSession(context.Background(), connect.NewRequest(&apiv1.GetEditorialSessionRequest{Id: sess.Id}))
		require.NoError(t, err)
		var out []string
		for _, c := range res.Msg.Markers[0].ContributorList {
			out = append(out, fmt.Sprintf("%t %d %s", c.U26, c.Age, c.Church))
		}
		return out
	}

	assert.Equal(t, []string{"true 20 Church One", "false 40 Church Two", "false 0 ", "false 0 ", "false 0 "}, details())

	for _, perms := range []string{`{"editor":true}`, `{"enabled":true,"members":true}`} {
		actAs(t, perms)
		assert.Equal(t, []string{"true 0 ", "false 0 ", "false 0 ", "false 0 ", "false 0 "}, details(), perms)
	}

	// The preview proxy needs none of it.
	got, err := api.loadSession(context.Background(), sess.Id, "")
	require.NoError(t, err)
	assert.False(t, got.Markers[0].ContributorList[0].U26)
}

func TestSearchMembersAddsDetails(t *testing.T) {
	log.ConfigureGlobalLogger(zerolog.Disabled)
	api := newTestEditorialAPI(t)
	fake := detailsFake()
	fake.people = []members.Person{
		{PersonID: "90005", DisplayName: "Ola Olsen", BirthDate: yearsAgo(19), Church: "Church Five"},
		{PersonID: "90006", DisplayName: "Ola Berg"},
	}
	api.members = fake
	api.people = newMemberDetails(fake)
	sess := createTestSession(t, api, "VX-1", "Søndag")
	saveTestSession(t, api, sess.Id, "Søndag", []*apiv1.EditorialMarker{{
		ContributorList: []*apiv1.EditorialContributor{{Name: "Ola Nordmann", PersonId: "90001"}},
	}})

	got, err := searchMembers(t, api, "ola", 0)
	require.NoError(t, err)
	var details []string
	for _, m := range got {
		details = append(details, fmt.Sprintf("%s %d %s", m.PersonId, m.Age, m.Church))
	}
	assert.Equal(t, []string{"90001 20 Church One", "90005 19 Church Five", "90006 0 "}, details)
	assert.Equal(t, 1, fake.getCount("90001"), "a linked contributor is looked up")
	assert.Zero(t, fake.getCount("90005"), "a search result brings its details")
}

// Yes when a linked contributor is U26, No when none is, empty when no one is
// linked: a name typed by hand has no known age.
func TestEditorialWorkbookU26Column(t *testing.T) {
	sess := editorial.Session{Vxid: "VX-1", Kind: editorial.KindRecording}
	marker := func(list ...contributor) markerRow {
		return markerRow{Marker: editorial.Marker{EndMs: 60_000}, Contributors: list}
	}
	markers := []markerRow{
		marker(contributor{Name: "Ola Nordmann", PersonID: "90001"}, contributor{Name: "Kari Nordmann", PersonID: "90002"}, contributor{Name: "Per Hansen"}),
		marker(contributor{Name: "Kari Nordmann", PersonID: "90002"}),
		marker(contributor{Name: "Per Hansen"}),
		marker(),
	}
	data, err := editorialWorkbook(sess, markers, map[string]bool{"90001": true}, exportLabelsByLanguage["en"])
	require.NoError(t, err)
	f, sheet := openWorkbook(t, data)
	var got []string
	for row := 1; row <= 5; row++ {
		v, err := f.GetCellValue(sheet, fmt.Sprintf("F%d", row))
		require.NoError(t, err)
		got = append(got, v)
	}
	assert.Equal(t, []string{"U26", "Yes", "No", "", ""}, got)
}
