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

// yearsAgo is a birth date that many years before today.
func yearsAgo(years int) time.Time {
	y, m, d := time.Now().AddDate(-years, 0, 0).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// Made-up people: uid(90001) is 20, uid(90002) is 40, uid(90003) has no
// birth date or church.
func detailsFake() *fakeMembers {
	return &fakeMembers{byID: map[string]members.Person{
		uid(90001): {UID: uid(90001), DisplayName: "Ola Nordmann", BirthDate: yearsAgo(20), Church: "Church One"},
		uid(90002): {UID: uid(90002), DisplayName: "Kari Nordmann", BirthDate: yearsAgo(40), Church: "Church Two"},
		uid(90003): {UID: uid(90003), DisplayName: "Per Hansen"},
	}}
}

// U26 for anyone; age and church only for those who may look up Members.
func TestGetEditorialSessionAddsDetails(t *testing.T) {
	log.ConfigureGlobalLogger(zerolog.Disabled)
	api := newTestEditorialAPI(t)
	api.people = members.NewCache(detailsFake())
	sess := createTestSession(t, api, "VX-1", "Søndag")
	saveTestSession(t, api, sess.Id, "Søndag", []*apiv1.EditorialMarker{{
		Name: "Tale", StartMs: 0, EndMs: 60_000,
		ContributorList: []*apiv1.EditorialContributor{
			{Name: "Ola Nordmann", MemberUid: uid(90001)},
			{Name: "Kari Nordmann", MemberUid: uid(90002)},
			{Name: "Per Hansen", MemberUid: uid(90003)},
			{Name: "Ola Nordmann"},
			// a client can't set them
			{Name: "Ola Olsen", MemberUid: uid(90009), Age: 18, Church: "Made Up", U26: true},
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
		{UID: uid(90005), DisplayName: "Ola Olsen", BirthDate: yearsAgo(19), Church: "Church Five"},
		{UID: uid(90006), DisplayName: "Ola Berg"},
	}
	api.members = fake
	api.people = members.NewCache(fake)
	sess := createTestSession(t, api, "VX-1", "Søndag")
	saveTestSession(t, api, sess.Id, "Søndag", []*apiv1.EditorialMarker{{
		ContributorList: []*apiv1.EditorialContributor{{Name: "Ola Nordmann", MemberUid: uid(90001)}},
	}})

	got, err := searchMembers(t, api, "ola", 0)
	require.NoError(t, err)
	var details []string
	for _, m := range got {
		details = append(details, fmt.Sprintf("%s %d %s", m.DisplayName, m.Age, m.Church))
	}
	assert.Equal(t, []string{"Ola Nordmann 20 Church One", "Ola Olsen 19 Church Five", "Ola Berg 0 "}, details)
	assert.Equal(t, 1, fake.getCount(uid(90001)), "a linked contributor is looked up")
	assert.Zero(t, fake.getCount(uid(90005)), "a search result brings its details")
	_, err = api.people.Get(context.Background(), []string{uid(90005)})
	require.NoError(t, err)
	assert.Equal(t, 1, fake.getCount(uid(90005)), "and isn't kept")
}

// Yes when a linked contributor is U26, No when none is, empty when no one is
// linked: a name typed by hand has no known age.
func TestEditorialWorkbookU26Column(t *testing.T) {
	sess := editorial.Session{Vxid: "VX-1", Kind: editorial.KindRecording}
	marker := func(list ...editorial.Contributor) editorial.MarkerRow {
		return editorial.MarkerRow{Marker: editorial.Marker{EndMs: 60_000}, Contributors: list}
	}
	markers := []editorial.MarkerRow{
		marker(editorial.Contributor{Name: "Ola Nordmann", MemberUID: uid(90001)}, editorial.Contributor{Name: "Kari Nordmann", MemberUID: uid(90002)}, editorial.Contributor{Name: "Per Hansen"}),
		marker(editorial.Contributor{Name: "Kari Nordmann", MemberUID: uid(90002)}),
		marker(editorial.Contributor{Name: "Per Hansen"}),
		marker(),
	}
	data, err := editorialWorkbook(sess, markers, map[string]bool{uid(90001): true}, exportLabelsByLanguage["en"])
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

// Ages and U26 are as of the recording: someone 28 today was 23 at a
// recording five years ago.
func TestMemberDetailsAsOfRecording(t *testing.T) {
	log.ConfigureGlobalLogger(zerolog.Disabled)
	api := newTestEditorialAPI(t)
	fake := &fakeMembers{byID: map[string]members.Person{
		uid(90007): {UID: uid(90007), DisplayName: "Ola Dahl", BirthDate: yearsAgo(28), Church: "Church Seven"},
	}, people: []members.Person{
		{UID: uid(90007), DisplayName: "Ola Dahl", BirthDate: yearsAgo(28), Church: "Church Seven"},
	}}
	api.members = fake
	api.people = members.NewCache(fake)
	sess := createTestSession(t, api, "VX-1", "Søndag")
	saveTestSession(t, api, sess.Id, "Søndag", []*apiv1.EditorialMarker{{
		Name: "Tale", StartMs: 0, EndMs: 60_000,
		ContributorList: []*apiv1.EditorialContributor{{Name: "Ola Dahl", MemberUid: uid(90007)}},
	}})
	recorded := time.Now().AddDate(-5, 0, 0).UnixMilli()
	_, err := api.db.Exec(`UPDATE sessions SET recording_start_ms = ? WHERE id = ?`, recorded, sess.Id)
	require.NoError(t, err)

	res, err := api.GetEditorialSession(context.Background(), connect.NewRequest(&apiv1.GetEditorialSessionRequest{Id: sess.Id}))
	require.NoError(t, err)
	c := res.Msg.Markers[0].ContributorList[0]
	assert.True(t, c.U26)
	assert.Equal(t, int32(23), c.Age)

	got, err := api.SearchMembers(context.Background(), connect.NewRequest(&apiv1.SearchMembersRequest{Query: "ola", SessionId: sess.Id}))
	require.NoError(t, err)
	require.NotEmpty(t, got.Msg.Members)
	assert.Equal(t, int32(23), got.Msg.Members[0].Age, "search ages as of the session")

	got, err = api.SearchMembers(context.Background(), connect.NewRequest(&apiv1.SearchMembersRequest{Query: "ola"}))
	require.NoError(t, err)
	assert.Equal(t, int32(28), got.Msg.Members[0].Age, "no session is today")
}

// The live clock start comes first, then a hand-entered window, then the day
// the session was made.
func TestRecordingDate(t *testing.T) {
	api := newTestEditorialAPI(t)
	day := func(s string) int64 {
		d, err := time.ParseInLocation("2006-01-02 15:04", s, meetingsTZ())
		require.NoError(t, err)
		return d.UnixMilli()
	}
	date := func(sess editorial.Session) string { return api.recordingDate(sess).Format("2006-01-02") }

	assert.Equal(t, "2020-03-01", date(editorial.Session{ClockStartMs: day("2020-03-01 10:00"), RecordingStartMs: day("2021-01-01 10:00"), CreatedAt: day("2022-01-01 10:00")}))
	assert.Equal(t, "2021-01-01", date(editorial.Session{Vxid: "VX-1", RecordingStartMs: day("2021-01-01 10:00"), CreatedAt: day("2022-01-01 10:00")}))
	assert.Equal(t, "2022-01-01", date(editorial.Session{Vxid: "VX-1", CreatedAt: day("2022-01-01 10:00")}), "no Vidispine")
	// Just after midnight in Oslo is still the previous day in UTC.
	assert.Equal(t, "2022-06-02", date(editorial.Session{ClockStartMs: day("2022-06-02 00:30")}))
}
