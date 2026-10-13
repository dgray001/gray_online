package mapgen

import "fmt"

// The space with the smallest farthest-space distance, lowest key on ties; the origin on a hexagon
func shapeCenter(ctx *mapScriptContext) Space {
	spaces := ctx.allSpaces()
	var center Space
	best := 0
	for _, space := range spaces {
		farthest := 0
		for _, other := range spaces {
			farthest = max(farthest, ctx.board.Distance(space, other))
		}
		if center == nil || farthest < best || (farthest == best && space.Key() < center.Key()) {
			center, best = space, farthest
		}
	}
	return center
}

// Each space's ring distance: hops from the shape's center, or steps in from its edge
func ringDistances(ctx *mapScriptContext, from string) (map[uint]int, error) {
	switch from {
	case "", "center":
		center := shapeCenter(ctx)
		distances := make(map[uint]int)
		for _, space := range ctx.allSpaces() {
			distances[space.Key()] = ctx.board.Distance(center, space)
		}
		return distances, nil
	case "edge":
		return ctx.edgeInsets(), nil
	}
	return nil, fmt.Errorf("blob ring from must be \"center\" or \"edge\", got %q", from)
}

// Steps inward from the outermost spaces (those missing a neighbor), so the map boundary is 0
func edgeInsets(spaces []Space) map[uint]int {
	insets := make(map[uint]int, len(spaces))
	queue := make([]Space, 0, len(spaces))
	for _, space := range spaces {
		if len(space.SortedAdjacent()) < 6 {
			insets[space.Key()] = 0
			queue = append(queue, space)
		}
	}
	for len(queue) > 0 {
		space := queue[0]
		queue = queue[1:]
		for _, adj := range space.SortedAdjacent() {
			if _, seen := insets[adj.Key()]; !seen {
				insets[adj.Key()] = insets[space.Key()] + 1
				queue = append(queue, adj)
			}
		}
	}
	return insets
}
