package risq

import (
	"maps"
	"slices"
)

func (m *gameMetrics) combatSpaces(turn uint16) map[*RisqSpace]bool {
	spaces := make(map[*RisqSpace]bool)
	for _, a := range m.actors {
		if a.attackTarget == nil || (a.space != a.attackTarget && !slices.Contains(a.space.sortedAdjacentSpaces(), a.attackTarget)) {
			continue
		}
		for _, target := range m.actors {
			if target.space == a.attackTarget && target.base.player_id != a.base.player_id {
				spaces[a.attackTarget] = true
				break
			}
		}
	}
	for _, h := range m.hits {
		if h.damage > 0 {
			spaces[h.attacker.space], spaces[h.target.space] = true, true
		}
	}
	for space := range m.lastSpaces {
		if m.spaceTurns[space]+1 < turn {
			continue
		}
		players := make(map[int]bool)
		for _, a := range m.actors {
			if a.space == space {
				players[a.base.player_id] = true
			}
		}
		if len(players) > 1 {
			spaces[space] = true
		}
	}
	return spaces
}

func combatGroups(spaces map[*RisqSpace]bool) [][]*RisqSpace {
	remaining := slices.Collect(maps.Keys(spaces))
	slices.SortFunc(remaining, func(a, b *RisqSpace) int { return int(a.coordinate_key) - int(b.coordinate_key) })
	groups := make([][]*RisqSpace, 0)
	for _, first := range remaining {
		if !spaces[first] {
			continue
		}
		group := []*RisqSpace{first}
		spaces[first] = false
		for i := 0; i < len(group); i++ {
			for _, adjacent := range group[i].sortedAdjacentSpaces() {
				if spaces[adjacent] {
					spaces[adjacent] = false
					group = append(group, adjacent)
				}
			}
		}
		groups = append(groups, group)
	}
	return groups
}
