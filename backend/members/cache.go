package members

import (
	"context"
	"slices"
	"sync"
	"time"
)

// Cache keeps people looked up by uid, so pages showing the same people don't
// ask Members each time.
type Cache struct {
	searcher Searcher
	keep time.Duration
	wait time.Duration

	mu    sync.Mutex
	byUID map[string]cachedPerson
}

// cachedPerson is a zero Person when Members doesn't know the uid.
type cachedPerson struct {
	Person
	expires time.Time
}

// NewCache keeps people for a day. It is nil, knowing no one, without a
// searcher.
func NewCache(s Searcher) *Cache {
	if s == nil {
		return nil
	}
	return &Cache{searcher: s, keep: 24 * time.Hour, wait: 2 * time.Second, byUID: map[string]cachedPerson{}}
}

// Get returns the people of uids, by uid. Those not kept are looked up
// together; when that fails or takes too long, they're left out with the
// error, and asked for again next time.
func (c *Cache) Get(ctx context.Context, uids []string) (map[string]Person, error) {
	if c == nil || len(uids) == 0 {
		return nil, nil
	}
	err := c.lookUp(ctx, uids)
	c.mu.Lock()
	defer c.mu.Unlock()
	out := map[string]Person{}
	for _, uid := range uids {
		if p, ok := c.byUID[uid]; ok && p.UID != "" {
			out[uid] = p.Person
		}
	}
	return out, err
}

// lookUp asks Members for the uids not kept. Anyone kept longer than keep is
// dropped first, whether asked for or not.
func (c *Cache) lookUp(ctx context.Context, uids []string) error {
	now := time.Now()
	c.mu.Lock()
	for uid, p := range c.byUID {
		if now.After(p.expires) {
			delete(c.byUID, uid)
		}
	}
	var missing []string
	for _, uid := range uids {
		if _, ok := c.byUID[uid]; !ok && !slices.Contains(missing, uid) {
			missing = append(missing, uid)
		}
	}
	c.mu.Unlock()
	if len(missing) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, c.wait)
	defer cancel()
	people, err := c.searcher.GetMany(ctx, missing)
	if err != nil {
		return err
	}
	expires := time.Now().Add(c.keep)
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, uid := range missing {
		c.byUID[uid] = cachedPerson{expires: expires}
	}
	for _, p := range people {
		c.byUID[p.UID] = cachedPerson{Person: p, expires: expires}
	}
	return nil
}
