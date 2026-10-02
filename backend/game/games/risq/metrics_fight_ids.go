package risq

func (m *gameMetrics) fightRoot(id int) int {
	for m.parents[id] != id {
		id = m.parents[id]
	}
	return id
}

func (m *gameMetrics) fightId(spaces map[*RisqSpace]bool, turn uint16) int {
	ids := make(map[int]bool)
	remember := func(id int) {
		if id > 0 {
			id = m.fightRoot(id)
			if m.lastTurns[id]+1 >= turn {
				ids[id] = true
			}
		}
	}
	for space := range spaces {
		if m.spaceTurns[space]+1 >= turn {
			remember(m.lastSpaces[space])
		}
		for _, adjacent := range space.adjacent_spaces {
			if m.spaceTurns[adjacent]+1 >= turn {
				remember(m.lastSpaces[adjacent])
			}
		}
	}
	if len(m.parents) == 0 {
		m.parents = append(m.parents, 0)
	}
	id := len(m.parents)
	for prior := range ids {
		id = min(id, prior)
	}
	if id == len(m.parents) {
		m.parents = append(m.parents, id)
	}
	for prior := range ids {
		m.parents[prior] = id
	}
	if m.lastTurns == nil {
		m.lastTurns = make(map[int]uint16)
		m.lastSpaces = make(map[*RisqSpace]int)
		m.spaceTurns = make(map[*RisqSpace]uint16)
	}
	m.lastTurns[id] = turn
	for space := range spaces {
		m.lastSpaces[space] = id
		m.spaceTurns[space] = turn
	}
	return id
}

func (r *GameRisq) Metrics() GameMetrics {
	out := GameMetrics{Turns: r.metrics.turns}
	indexes := make(map[int]int)
	for _, sample := range r.metrics.frames {
		sample.FightId = r.metrics.fightRoot(sample.FightId)
		index, found := indexes[sample.FightId]
		if !found {
			index = len(out.Fights)
			indexes[sample.FightId] = index
			out.Fights = append(out.Fights, FightMetrics{FightId: sample.FightId})
		}
		out.Fights[index].Ticks = append(out.Fights[index].Ticks, sample)
	}
	return out
}
