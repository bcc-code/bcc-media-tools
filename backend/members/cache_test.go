package members

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func date(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse(time.DateOnly, s)
	require.NoError(t, err)
	return d
}

func TestPersonAge(t *testing.T) {
	born := func(s string) Person { return Person{BirthDate: date(t, s)} }
	today := date(t, "2026-11-12")
	assert.Equal(t, 25, born("2000-11-13").Age(today), "26 tomorrow")
	assert.Equal(t, 26, born("2000-11-12").Age(today), "26 today")
	assert.Equal(t, 36, born("1990-01-01").Age(today))
	assert.Zero(t, Person{}.Age(today), "unknown")

	leap := born("2000-02-29")
	assert.Equal(t, 25, leap.Age(date(t, "2026-02-28")))
	assert.Equal(t, 26, leap.Age(date(t, "2026-03-01")), "26 on 1 March")

	// The date where today is counts, not the time of day.
	oslo, err := time.LoadLocation("Europe/Oslo")
	require.NoError(t, err)
	assert.Equal(t, 25, born("2000-11-13").Age(time.Date(2026, 11, 12, 23, 59, 0, 0, oslo)))

	assert.True(t, born("2000-11-13").IsU26(today))
	assert.False(t, born("2000-11-12").IsU26(today))
	assert.False(t, Person{}.IsU26(today), "unknown")
}

// fakeSearcher knows the people in byUID.
type fakeSearcher struct {
	byUID map[string]Person
	err   error
	// each GetMany takes this long
	delay time.Duration

	mu   sync.Mutex
	gets map[string]int
}

func (f *fakeSearcher) Search(context.Context, string, int) ([]Person, error) { return nil, nil }

func (f *fakeSearcher) GetMany(ctx context.Context, uids []string) ([]Person, error) {
	f.mu.Lock()
	if f.gets == nil {
		f.gets = map[string]int{}
	}
	for _, uid := range uids {
		f.gets[uid]++
	}
	f.mu.Unlock()
	select {
	case <-time.After(f.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if f.err != nil {
		return nil, f.err
	}
	var out []Person
	for _, uid := range uids {
		if p, ok := f.byUID[uid]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakeSearcher) getCount(uid string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.gets[uid]
}

func cacheFake() *fakeSearcher {
	return &fakeSearcher{byUID: map[string]Person{
		"u1": {UID: "u1", DisplayName: "Ola Nordmann", Church: "Church One"},
		"u2": {UID: "u2", DisplayName: "Kari Nordmann", Church: "Church Two"},
		"u3": {UID: "u3", DisplayName: "Per Hansen"},
	}}
}

func TestCacheLooksEachUpOnce(t *testing.T) {
	fake := cacheFake()
	c := NewCache(fake)
	ctx := context.Background()

	uids := []string{"u1", "u2", "u3", "u9"}
	for range 2 {
		got, err := c.Get(ctx, uids)
		require.NoError(t, err)
		assert.Equal(t, []string{"Church One", "Church Two", ""}, []string{got["u1"].Church, got["u2"].Church, got["u3"].Church})
		assert.NotContains(t, got, "u9", "unknown to Members")
	}
	for _, uid := range uids {
		assert.Equal(t, 1, fake.getCount(uid), "%s looked up once, unknown ones too", uid)
	}

	// After a day everyone is dropped, those not asked for too; a failed
	// lookup then finds no one rather than the old details.
	for uid, p := range c.byUID {
		p.expires = time.Now().Add(-time.Minute)
		c.byUID[uid] = p
	}
	fake.err = errors.New("members down")
	got, err := c.Get(ctx, []string{"u1"})
	assert.Error(t, err)
	assert.Empty(t, got)
	assert.Empty(t, c.byUID)
	fake.err = nil
	got, err = c.Get(ctx, []string{"u1"})
	require.NoError(t, err)
	assert.Equal(t, "Church One", got["u1"].Church)
	assert.Equal(t, 3, fake.getCount("u1"))

	got, err = NewCache(nil).Get(ctx, uids)
	assert.NoError(t, err)
	assert.Nil(t, got, "no Members, no one known")
}

// A slow Members gives no one rather than a slow page, and is asked again
// next time.
func TestCacheDoesNotWaitLong(t *testing.T) {
	fake := cacheFake()
	fake.delay = time.Second
	c := NewCache(fake)
	c.wait = 20 * time.Millisecond

	start := time.Now()
	got, err := c.Get(context.Background(), []string{"u1", "u2"})
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Empty(t, got)
	assert.Less(t, time.Since(start), 500*time.Millisecond)

	fake.delay = 0
	got, err = c.Get(context.Background(), []string{"u1"})
	require.NoError(t, err)
	assert.Contains(t, got, "u1")
	assert.Equal(t, 2, fake.getCount("u1"))
}
