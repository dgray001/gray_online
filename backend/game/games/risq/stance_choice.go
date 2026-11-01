package risq

type stancePick struct {
	unit       *RisqUnit
	candidates []Attackable
}

func chooseStanceTarget(pick stancePick, counts map[Attackable]int) Attackable {
	stack := int(pick.unit.attack_stack)
	if stack == 0 {
		return pick.candidates[0]
	}
	least := pick.candidates[0]
	for _, candidate := range pick.candidates {
		if counts[candidate] < stack {
			return candidate
		}
		if counts[candidate] < counts[least] {
			least = candidate
		}
	}
	return least
}
