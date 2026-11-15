package backoffice_handlers

import (
	"crypto/sha256"
	"sync"
	"time"
)

// rotationGraceTTL is how long a rotated-out backoffice refresh token still
// resolves to the token that replaced it.
const rotationGraceTTL = 30 * time.Second

// rotationGrace remembers, briefly and in memory, which refresh token replaced
// which. Backoffice admins keep a single refresh token (one column, no
// session table), so without it two tabs refreshing at once would race: the
// loser presents a token that was just rotated out and gets signed out. Keys
// are hashes, so the map never holds a presented token in the clear.
type rotationGrace struct {
	mu      sync.Mutex
	entries map[[32]byte]graceEntry
}

type graceEntry struct {
	successor string
	until     time.Time
}

func newRotationGrace() *rotationGrace {
	return &rotationGrace{entries: map[[32]byte]graceEntry{}}
}

func (g *rotationGrace) remember(old, successor string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	for k, e := range g.entries {
		if now.After(e.until) {
			delete(g.entries, k)
		}
	}
	g.entries[sha256.Sum256([]byte(old))] = graceEntry{successor: successor, until: now.Add(rotationGraceTTL)}
}

func (g *rotationGrace) successor(old string) (string, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	e, ok := g.entries[sha256.Sum256([]byte(old))]
	if !ok || time.Now().After(e.until) {
		return "", false
	}
	return e.successor, true
}
