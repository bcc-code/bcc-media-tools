package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/members"
	"bcc-media-tools/playout"
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/bcc-code/mediabank-bridge/log"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// playoutEvents serves a manifest per event and counts the reads.
type playoutEvents struct {
	events    []playout.Event
	manifests map[string][]playout.ManifestEntry
	// reads of an event refused before it answers; -1 always refuses
	refuse map[string]int

	mu    sync.Mutex
	reads map[string]int
}

func (p *playoutEvents) ListEvents(context.Context) ([]playout.Event, error) { return p.events, nil }

func (p *playoutEvents) GetManifest(_ context.Context, id string, _ ...string) (*playout.Manifest, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.reads == nil {
		p.reads = map[string]int{}
	}
	p.reads[id]++
	if n := p.refuse[id]; n < 0 || p.reads[id] <= n {
		return nil, errors.New("unexpected status 429 Too Many Requests")
	}
	return &playout.Manifest{EventID: id, Entries: p.manifests[id]}, nil
}

// noSpeakerWaits drops the pauses between Playout reads for the test.
func noSpeakerWaits(t *testing.T) {
	pause, retry := speakersPause, speakersRetryAfter
	speakersPause, speakersRetryAfter = 0, 0
	t.Cleanup(func() { speakersPause, speakersRetryAfter = pause, retry })
}

func speakerAt(t *testing.T, name, personID, at string) playout.ManifestEntry {
	return speakerWithData(t, name, fmt.Sprintf(`{"personId":%s,"name":%q}`, personID, name), at)
}

func TestFrequentSpeakersCountsEventsAndSkipsPlaceholders(t *testing.T) {
	log.ConfigureGlobalLogger(zerolog.Disabled)
	noSpeakerWaits(t)
	p := &playoutEvents{
		events: []playout.Event{{ID: "1", Status: "archived"}, {ID: "2", Status: "archived"}, {ID: "3", Status: "active"}},
		manifests: map[string][]playout.ManifestEntry{
			// Kari speaks at two events; Ola's name shows five times at one.
			"1": {
				speakerAt(t, "Kari Nordmann", `"90002"`, "2026-01-10T10:00:00Z"),
				speakerAt(t, "Ola Nordmann", "90001", "2026-01-10T10:05:00Z"),
				speakerAt(t, "Ola Nordmann", "90001", "2026-01-10T10:06:00Z"),
				speakerAt(t, "Ola Nordmann", "90001", "2026-01-10T10:07:00Z"),
				speakerAt(t, "Ola Nordmann", "90001", "2026-01-10T10:08:00Z"),
				speakerAt(t, "Ola Nordmann", "90001", "2026-01-10T10:09:00Z"),
				speakerAt(t, "Per Hansen", "1337", "2026-01-10T10:10:00Z"),
				songEntry(t, "HV 1 - Test song", "2026-01-10T10:20:00Z"),
			},
			"2": {speakerAt(t, "Kari Nordmann-Berg", `"90002"`, "2026-02-10T10:00:00Z")},
			"3": {speakerAt(t, "Åse Testesen", `"90004"`, "2026-03-10T10:00:00Z")},
		},
	}
	f := newFrequentSpeakers(p)
	require.NoError(t, f.refresh(context.Background()))

	all := f.match("", 10)
	require.Len(t, all, 3, "the placeholder is not a person")
	assert.Equal(t, "90002", all[0].PersonID, "two events beat five entries at one")
	assert.Equal(t, "Kari Nordmann-Berg", all[0].Name, "the latest name")
	assert.Equal(t, 2, all[0].Events)
	assert.Equal(t, "90001", all[1].PersonID)
	assert.Equal(t, 5, all[1].Entries)
	assert.Equal(t, 1, f.appearances("90004"))
	assert.Zero(t, f.appearances("90009"))

	got := f.match("ase", 10)
	require.Len(t, got, 1, "accents don't matter")
	assert.Equal(t, "Åse Testesen", got[0].Name)
	assert.Len(t, f.match("nordmann", 1), 1, "limit")
	assert.Empty(t, f.match("nordmann olsen", 10), "every word must match")

	// Archived events are read once; the active one again.
	require.NoError(t, f.refresh(context.Background()))
	assert.Equal(t, map[string]int{"1": 1, "2": 1, "3": 2}, p.reads)
}

func TestFrequentSpeakersRetriesRefusedEvents(t *testing.T) {
	log.ConfigureGlobalLogger(zerolog.Disabled)
	noSpeakerWaits(t)
	p := &playoutEvents{
		events: []playout.Event{{ID: "1", Status: "archived"}, {ID: "2", Status: "archived"}, {ID: "3", Status: "archived"}},
		manifests: map[string][]playout.ManifestEntry{
			"1": {speakerAt(t, "Ola Nordmann", `"90001"`, "2026-01-10T10:00:00Z")},
			"2": {speakerAt(t, "Kari Nordmann", `"90002"`, "2026-02-10T10:00:00Z")},
			"3": {speakerAt(t, "Per Hansen", `"90005"`, "2026-03-10T10:00:00Z")},
		},
		// 2 answers on its second read; 3 not at all
		refuse: map[string]int{"2": 1, "3": -1},
	}
	f := newFrequentSpeakers(p)
	require.NoError(t, f.refresh(context.Background()))
	assert.Len(t, f.match("", 10), 2, "the retry reads event 2")
	assert.Equal(t, map[string]int{"1": 1, "2": 2, "3": 1 + speakersRetries}, p.reads)

	// An event never read isn't kept as read: the next refresh tries it again.
	p.refuse = nil
	require.NoError(t, f.refresh(context.Background()))
	assert.Len(t, f.match("", 10), 3)
	assert.Equal(t, map[string]int{"1": 1, "2": 2, "3": 2 + speakersRetries}, p.reads)
}

func TestSearchMembersPutsFrequentSpeakersFirst(t *testing.T) {
	log.ConfigureGlobalLogger(zerolog.Disabled)
	noSpeakerWaits(t)
	api := newTestEditorialAPI(t)
	api.speakers = newFrequentSpeakers(&playoutEvents{
		events: []playout.Event{{ID: "1", Status: "archived"}, {ID: "2", Status: "archived"}},
		manifests: map[string][]playout.ManifestEntry{
			"1": {speakerAt(t, "Ola Nordmann", `"90001"`, "2026-01-10T10:00:00Z"), speakerAt(t, "Kari Nordmann", `"90002"`, "2026-01-10T10:10:00Z")},
			"2": {speakerAt(t, "Kari Nordmann", `"90002"`, "2026-02-10T10:00:00Z")},
		},
	})
	require.NoError(t, api.speakers.refresh(context.Background()))
	api.members = &fakeMembers{people: []members.Person{
		{PersonID: "90003", DisplayName: "Ola Nordvik"},
		{PersonID: "90001", DisplayName: "Ola Nordmann"},
	}}
	type member struct {
		ID, Name    string
		Appearances int32
	}
	ids := func(ms []*apiv1.Member) []member {
		var out []member
		for _, m := range ms {
			out = append(out, member{m.PersonId, m.DisplayName, m.Appearances})
		}
		return out
	}

	got, err := searchMembers(t, api, "", 0)
	require.NoError(t, err)
	assert.Equal(t, []member{{"90002", "Kari Nordmann", 2}, {"90001", "Ola Nordmann", 1}}, ids(got), "an empty field lists the frequent")

	got, err = searchMembers(t, api, "ola", 0)
	require.NoError(t, err)
	assert.Equal(t, []member{{"90001", "Ola Nordmann", 1}, {"90003", "Ola Nordvik", 0}}, ids(got), "frequent first, Members after, no repeats")

	got, err = searchMembers(t, api, "o", 0)
	require.NoError(t, err)
	assert.Equal(t, []member{{"90001", "Ola Nordmann", 1}}, ids(got), "one letter finds only the frequent")

	api.members = &fakeMembers{err: errors.New("members down")}
	got, err = searchMembers(t, api, "kari", 0)
	require.NoError(t, err)
	assert.Equal(t, []member{{"90002", "Kari Nordmann", 2}}, ids(got), "frequent speakers without Members")
}

func TestFoldTreatsAaAsÅ(t *testing.T) {
	assert.Equal(t, "asen", fold("Åsen"))
	assert.Equal(t, "asen", fold("Aasen"))
	assert.Equal(t, "odegard", fold("Ødegård"))
	assert.Equal(t, "odegard", fold("Ødegaard"))

	f := &frequentSpeakers{}
	f.people, f.byID = rankSpeakers(map[string]map[string]eventSpeaker{"1": {
		"90006": {Name: "Kari Åsen", Entries: 1},
		"90007": {Name: "Ola Aasheim", Entries: 1},
	}})
	var names []string
	for _, p := range f.match("aas", 10) {
		names = append(names, p.Name)
	}
	assert.ElementsMatch(t, []string{"Kari Åsen", "Ola Aasheim"}, names, "either spelling finds both")
	assert.Len(t, f.match("ås", 10), 2)
}

// Aadalen and Ådalen are different names: both are found, the one spelt as
// typed first, whoever speaks more often.
func TestMatchPutsTheSpellingTypedFirst(t *testing.T) {
	f := &frequentSpeakers{}
	f.people, f.byID = rankSpeakers(map[string]map[string]eventSpeaker{
		"1": {"90008": {Name: "Kari Ådalen", Entries: 5}, "90009": {Name: "Ola Aadalen", Entries: 1}},
		"2": {"90008": {Name: "Kari Ådalen", Entries: 1}},
	})
	names := func(query string, limit int) []string {
		var out []string
		for _, p := range f.match(query, limit) {
			out = append(out, p.Name)
		}
		return out
	}
	assert.Equal(t, []string{"Ola Aadalen", "Kari Ådalen"}, names("aadalen", 10))
	assert.Equal(t, []string{"Kari Ådalen", "Ola Aadalen"}, names("ådalen", 10))
	assert.Equal(t, []string{"Kari Ådalen", "Ola Aadalen"}, names("adalen", 10), "no å typed reads as å")
	assert.Equal(t, []string{"Ola Aadalen"}, names("aad", 1), "the limit keeps the spelling typed")
	assert.Equal(t, []string{"Kari Ådalen", "Ola Aadalen"}, names("", 10), "no query, by frequency")
}
