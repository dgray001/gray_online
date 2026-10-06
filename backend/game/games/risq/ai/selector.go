package ai

import (
	"fmt"
	"slices"
	"sort"
)

type relation uint8

const (
	relationAny relation = iota
	relationOwn
	relationEnemy
	relationUnowned
)

var relationNames = map[string]relation{"any": relationAny, "own": relationOwn, "enemy": relationEnemy, "unowned": relationUnowned}

// The owner and place filters every loop source shares: "owner" (own, enemy, unowned or any), "player_id" (a number or expression), "within" and "from"
type selector struct {
	relation relation
	player   *amount
	near     nearFilter
}

func parseSelector(where map[string]any, fallback relation) (selector, error) {
	s := selector{relation: fallback}
	if name, ok := where["owner"].(string); ok {
		r, known := relationNames[name]
		if !known {
			return s, fmt.Errorf("unknown owner %q", name)
		}
		s.relation = r
	}
	if raw, present := where["player_id"]; present {
		a, err := parseAmount(raw)
		if err != nil {
			return s, fmt.Errorf("player_id %v", err)
		}
		s.player = &a
	}
	var err error
	s.near, err = parseNearFilter(where)
	return s, err
}

// Whether something owned by owner (-1 for no one) at loc passes the owner, player and place filters
func (s selector) allows(view View, internals *Internals, owner int, loc ZoneRef) bool {
	me := view.PlayerID()
	switch s.relation {
	case relationOwn:
		if owner != me {
			return false
		}
	case relationEnemy:
		if owner < 0 || owner == me {
			return false
		}
	case relationUnowned:
		if owner >= 0 {
			return false
		}
	}
	if s.player != nil && owner != s.player.int(view, internals) {
		return false
	}
	return s.near.contains(view, internals, loc)
}

// Rejects any key of a where clause its source does not take, so a typo is an error and not a silently ignored filter
func checkKeys(where map[string]any, allowed ...string) error {
	keys := make([]string, 0, len(where))
	for key := range where {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !slices.Contains(allowed, key) {
			return fmt.Errorf("unknown where key %q", key)
		}
	}
	return nil
}
