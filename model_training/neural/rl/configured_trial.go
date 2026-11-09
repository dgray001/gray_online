package rl

import "fmt"

func configuredTrial(hidden int, start string, rate float64, entropy float64, seed int64) Trial {
	t := trial(start, rate, entropy, seed)
	t.Hidden = hidden
	t.ID = fmt.Sprintf("rl-%s-r%.4f-e%.2f-h%d-s%d", start, rate, entropy, hidden, seed)
	return t
}
