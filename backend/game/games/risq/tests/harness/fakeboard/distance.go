package fakeboard

func (b *Board) hopDistances(from *space) []uint16 {
	row := make([]uint16, len(b.order))
	for i := range row {
		row[i] = Unreachable
	}
	row[from.distanceIndex] = 0
	queue := make([]*space, 1, len(b.order))
	queue[0] = from
	for head := 0; head < len(queue); head++ {
		current := queue[head]
		visit := func(next *space) {
			if next != nil && row[next.distanceIndex] == Unreachable {
				row[next.distanceIndex] = row[current.distanceIndex] + 1
				queue = append(queue, next)
			}
		}
		for _, next := range current.borders {
			visit(next)
		}
		for _, next := range current.links {
			visit(next)
		}
	}
	return row
}
