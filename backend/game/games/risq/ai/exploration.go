package ai

type ObservationHistory map[Coordinate]int

func (h ObservationHistory) Observe(spaces []SpaceInfo, turn int) {
	for _, space := range spaces {
		if space.Vision >= 2 {
			h[space.Space] = turn
		}
	}
}
