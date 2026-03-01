package risq

type RisqUnitIntent struct {
	move *MoveIntent
}

type MoveIntent struct {
	path      []*RisqZone
	next_step *RisqZone
	// true if the next step is a move within a space
	intra_step bool
}
