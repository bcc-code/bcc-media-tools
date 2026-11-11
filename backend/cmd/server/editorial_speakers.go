package main

import (
	"bcc-media-tools/playout"
	"cmp"
	"context"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/bcc-code/mediabank-bridge/log"
	"golang.org/x/text/unicode/norm"
)

// Playout answers 429 to bursts: read slowly, retry refused. Vars for tests.
var (
	speakersRefreshEvery = 6 * time.Hour
	speakersPause        = 250 * time.Millisecond
	speakersRetryAfter   = time.Minute
	speakersRetries      = 3
)

// Archived manifests don't change: read once.
const playoutEventArchived = "archived"

// frequentSpeakers counts Playout speakers with a personID, in memory.
type frequentSpeakers struct {
	playout playoutClient
	// days are counted in the meetings' time zone
	loc *time.Location

	mu     sync.RWMutex
	people []frequentSpeaker
	byID   map[string]frequentSpeaker
	// by event ID
	archived map[string]map[string]eventSpeaker
}

type frequentSpeaker struct {
	PersonID string
	// latest name in Playout
	Name string
	// days spoken, the ranking; events and entries break ties
	Days, Events, Entries int
	// folded for matching; spelt keeps "aa" apart from "å"
	words, spelt []string
	lastSeen     time.Time
}

type eventSpeaker struct {
	Name    string
	Entries int
	// dates spoken, "2006-01-02"
	Days     map[string]bool
	LastSeen time.Time
}

func newFrequentSpeakers(p playoutClient) *frequentSpeakers {
	loc, err := time.LoadLocation(recordingTZ)
	if err != nil {
		loc = time.UTC
	}
	return &frequentSpeakers{playout: p, loc: loc, archived: map[string]map[string]eventSpeaker{}}
}

// run refreshes now and every speakersRefreshEvery, until ctx ends.
func (f *frequentSpeakers) run(ctx context.Context) {
	for {
		if err := f.refresh(ctx); err != nil {
			log.L.Info().Err(err).Msg("frequent speakers: could not read Playout events")
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(speakersRefreshEvery):
		}
	}
}

// refresh skips events still refused after the retries.
func (f *frequentSpeakers) refresh(ctx context.Context) error {
	events, err := f.playout.ListEvents(ctx)
	if err != nil {
		return err
	}
	f.mu.RLock()
	archived := f.archived
	f.mu.RUnlock()

	perEvent := map[string]map[string]eventSpeaker{}
	var pending []playout.Event
	for _, ev := range events {
		if counts, ok := archived[ev.ID]; ok {
			perEvent[ev.ID] = counts
		} else {
			pending = append(pending, ev)
		}
	}
	for try := 0; len(pending) > 0 && try <= speakersRetries; try++ {
		if try > 0 && !sleep(ctx, speakersRetryAfter) {
			return ctx.Err()
		}
		var refused []playout.Event
		for _, ev := range pending {
			if !sleep(ctx, speakersPause) {
				return ctx.Err()
			}
			manifest, err := f.playout.GetManifest(ctx, ev.ID, playout.ContentTypeSpeaker)
			if err != nil {
				refused = append(refused, ev)
				continue
			}
			perEvent[ev.ID] = countSpeakers(manifest.Entries, f.loc)
		}
		pending = refused
	}
	if len(pending) > 0 {
		log.L.Info().Int("events", len(pending)).Msg("frequent speakers: some Playout manifests could not be read")
	}

	nextArchived := map[string]map[string]eventSpeaker{}
	for _, ev := range events {
		if counts, ok := perEvent[ev.ID]; ok && ev.Status == playoutEventArchived {
			nextArchived[ev.ID] = counts
		}
	}
	people, byID := rankSpeakers(perEvent)

	f.mu.Lock()
	f.people, f.byID, f.archived = people, byID, nextArchived
	f.mu.Unlock()
	return nil
}

func countSpeakers(entries []playout.ManifestEntry, loc *time.Location) map[string]eventSpeaker {
	counts := map[string]eventSpeaker{}
	for _, entry := range entries {
		if entry.Type != playout.ContentTypeSpeaker {
			continue
		}
		s := playoutSpeaker(entry)
		if s.PersonID == "" || s.Name == "" {
			continue
		}
		c := counts[s.PersonID]
		c.Entries++
		if c.Days == nil {
			c.Days = map[string]bool{}
		}
		c.Days[entry.Timestamp.In(loc).Format(time.DateOnly)] = true
		if !entry.Timestamp.Before(c.LastSeen) {
			c.Name, c.LastSeen = s.Name, entry.Timestamp
		}
		counts[s.PersonID] = c
	}
	return counts
}

// rankSpeakers orders by days spoken: an event can be one meeting or a
// week-long conference, and a meeting repeats names.
func rankSpeakers(perEvent map[string]map[string]eventSpeaker) ([]frequentSpeaker, map[string]frequentSpeaker) {
	byID := map[string]frequentSpeaker{}
	days := map[string]map[string]bool{}
	for _, counts := range perEvent {
		for id, c := range counts {
			p := byID[id]
			p.PersonID = id
			p.Events++
			p.Entries += c.Entries
			if !c.LastSeen.Before(p.lastSeen) {
				p.Name, p.lastSeen = c.Name, c.LastSeen
			}
			byID[id] = p
			if days[id] == nil {
				days[id] = map[string]bool{}
			}
			for d := range c.Days {
				days[id][d] = true
			}
		}
	}
	people := make([]frequentSpeaker, 0, len(byID))
	for id, p := range byID {
		p.Days = len(days[id])
		p.words, p.spelt = nameWords(fold(p.Name)), nameWords(foldAccents(p.Name))
		byID[id] = p
		people = append(people, p)
	}
	slices.SortFunc(people, func(a, b frequentSpeaker) int {
		return cmp.Or(
			cmp.Compare(b.Days, a.Days),
			cmp.Compare(b.Events, a.Events),
			cmp.Compare(b.Entries, a.Entries),
			strings.Compare(a.Name, b.Name),
		)
	})
	return people, byID
}

// match: each query word starts a word of the name; empty matches all.
// Names spelt as typed (aa or å) come before the other spelling.
func (f *frequentSpeakers) match(query string, limit int) []frequentSpeaker {
	if f == nil {
		return nil
	}
	typed, typedSpelt := nameWords(fold(query)), nameWords(foldAccents(query))
	f.mu.RLock()
	defer f.mu.RUnlock()
	var spelt, other []frequentSpeaker
	for _, p := range f.people {
		if len(spelt) == limit {
			break
		}
		if !startsWords(p.words, typed) {
			continue
		}
		if startsWords(p.spelt, typedSpelt) {
			spelt = append(spelt, p)
		} else {
			other = append(other, p)
		}
	}
	out := append(spelt, other...)
	return out[:min(limit, len(out))]
}

// startsWords reports whether each of typed starts one of words.
func startsWords(words, typed []string) bool {
	return !slices.ContainsFunc(typed, func(w string) bool {
		return !slices.ContainsFunc(words, func(n string) bool { return strings.HasPrefix(n, w) })
	})
}

// nameWords splits at spaces, hyphens and dots ("kari nordmann-berg").
func nameWords(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || r == '-' || r == '.'
	})
}

func (f *frequentSpeakers) appearances(personID string) int {
	if f == nil {
		return 0
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.byID[personID].Days
}

// sleep reports false when ctx ends first.
func sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// fold is foldAccents with "aa" as "å": "Aasen" and "Åsen" → "asen".
func fold(s string) string {
	return strings.ReplaceAll(foldAccents(s), "aa", "a")
}

// foldAccents lowercases and strips accents: "Åsen" → "asen".
func foldAccents(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		switch {
		case unicode.Is(unicode.Mn, r):
		case r == 'ø':
			b.WriteRune('o')
		case r == 'æ':
			b.WriteString("ae")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
