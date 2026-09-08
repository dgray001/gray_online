package ai

func axialDistance(a, b Coordinate) int {
	dq := abs(a.X - b.X)
	dr := abs(a.Y - b.Y)
	ds := abs(a.X + a.Y - b.X - b.Y)
	return (dq + dr + ds) / 2
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func locationDistance(a, b ZoneRef) int {
	if a.Space == b.Space {
		return axialDistance(a.Zone, b.Zone)
	}
	return axialDistance(a.Space, b.Space) * 6
}
