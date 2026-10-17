package mapgen

import (
	"fmt"
	"math"
	"sort"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

func placeStartGroup(ctx *mapScriptContext, p playerStartsParams, starts []playerStartInfo, footprints [][]Space, area_size int, first_player int) ([]startResourceSlot, error) {
	slots, err := planStartResources(ctx, p.Resources, area_size)
	if err != nil {
		return nil, err
	}
	return slots, placeStartContents(ctx, p, starts, footprints, first_player)
}

func placeBalancedStarts(ctx *mapScriptContext, p playerStartsParams, starts []playerStartInfo, footprints [][]Space, area_size int) error {
	balance, err := p.startBalance(ctx.vars)
	if err != nil {
		return err
	}
	groups := 1
	if balance == defs.StartBalance_RANDOM {
		groups = len(starts)
	}
	group_slots := make([][]startResourceSlot, groups)
	group_bounds := func(group int) ([]playerStartInfo, [][]Space) {
		if balance == defs.StartBalance_RANDOM {
			return starts[group : group+1], footprints[group : group+1]
		}
		return starts, footprints
	}
	for group := range group_slots {
		group_starts, group_footprints := group_bounds(group)
		if group_slots[group], err = placeStartGroup(ctx, p, group_starts, group_footprints, area_size, group); err != nil {
			return err
		}
	}
	for group, slots := range group_slots {
		group_starts, group_footprints := group_bounds(group)
		if err := placeStartResources(ctx, group_starts, group_footprints, slots); err != nil {
			return err
		}
	}
	return nil
}

func resolveTrianglePlayerStarts(ctx *mapScriptContext, inset int) ([]playerStartInfo, error) {
	if ctx.shape != "triangle" {
		return nil, fmt.Errorf("triangle starts require a triangle shape")
	}
	edge := int(ctx.vars["edge_length"])
	innerEdge := edge - 3*inset
	if inset < 0 || innerEdge < 1 {
		return nil, fmt.Errorf("triangle inset %d leaves no perimeter", inset)
	}
	perimeter := 3 * innerEdge
	if ctx.num_players > perimeter {
		return nil, fmt.Errorf("triangle perimeter has %d spaces for %d players", perimeter, ctx.num_players)
	}
	offset := triangleCenterOffset(edge)
	low, high := inset-offset, edge-offset-2*inset
	corners := [3]game_utils.Coordinate2D{{X: low, Y: low}, {X: high, Y: low}, {X: low, Y: high}}
	directions := [3]game_utils.Coordinate2D{{X: 1}, {X: -1, Y: 1}, {Y: -1}}
	inward := [3]game_utils.Coordinate2D{{Y: 1}, {X: -1}, {X: 1, Y: -1}}
	first := ctx.rng.Intn(len(corners)) * innerEdge
	traversal := 1
	if ctx.rng.Intn(2) == 1 {
		traversal = -1
	}
	starts := make([]playerStartInfo, ctx.num_players)
	for i := range starts {
		position := (first + traversal*int(math.Round(float64(i*perimeter)/float64(ctx.num_players))) + perimeter) % perimeter
		side, step := position/innerEdge, position%innerEdge
		coordinate := game_utils.Coordinate2D{X: corners[side].X + step*directions[side].X, Y: corners[side].Y + step*directions[side].Y}
		space := ctx.board.Space(coordinate)
		if space == nil {
			return nil, fmt.Errorf("triangle perimeter space %v is missing", coordinate)
		}
		starts[i] = playerStartInfo{space: space, direction: inward[side]}
	}
	return starts, nil
}

// Places players on one perimeter ring
func ringStartDirection(c game_utils.Coordinate2D) game_utils.Coordinate2D {
	directions := game_utils.AxialDirectionVectors()
	best := directions[0]
	bestDot := c.X*best.X + c.Y*best.Y
	for _, direction := range directions[1:] {
		dot := c.X*direction.X + c.Y*direction.Y
		if dot > bestDot {
			best, bestDot = direction, dot
		}
	}
	return best
}

func resolveRingPlayerStarts(ctx *mapScriptContext, starting_distance int) ([]playerStartInfo, error) {
	if starting_distance < 1 {
		return nil, fmt.Errorf("ring player starts require a positive starting distance")
	}
	candidates := make([]Space, 0)
	origin := game_utils.Coordinate2D{}
	for _, space := range ctx.allSpaces() {
		if int(game_utils.AxialDistance(origin, space.Coordinate())) == starting_distance {
			candidates = append(candidates, space)
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("ring has no spaces at distance %d", starting_distance)
	}
	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i].Coordinate(), candidates[j].Coordinate()
		angleA := math.Atan2(float64(a.Y), float64(a.X))
		angleB := math.Atan2(float64(b.Y), float64(b.X))
		if angleA != angleB {
			return angleA < angleB
		}
		if a.X != b.X {
			return a.X < b.X
		}
		return a.Y < b.Y
	})
	// Rotate the evenly spaced pattern by the seeded RNG so maps retain orientation variation.
	offset := ctx.rng.Intn(len(candidates))
	starts := make([]playerStartInfo, ctx.num_players)
	for i := 0; i < ctx.num_players; i++ {
		index := (offset + (i*len(candidates))/ctx.num_players) % len(candidates)
		space := candidates[index]
		starts[i] = playerStartInfo{space: space, direction: ringStartDirection(space.Coordinate())}
	}
	return starts, nil
}

func rectangleSpaceBounds(spaces []Space) (row_min int, row_max int, col_min int, col_max int) {
	first := true
	for _, s := range spaces {
		row := s.Coordinate().Y
		col := s.Coordinate().X + floorDiv2(row)
		if first {
			row_min, row_max, col_min, col_max = row, row, col, col
			first = false
			continue
		}
		row_min, row_max = min(row_min, row), max(row_max, row)
		col_min, col_max = min(col_min, col), max(col_max, col)
	}
	return
}

func rowPlayerStartColumn(index int, n int, col_min int, col_max int) int {
	span := col_max - col_min
	return col_min + int(math.Round(float64(index*span)/float64(max(1, n-1))))
}

// A row's own column range among the spaces the current shape actually kept -- e.g. a triangle's
// rows narrow toward its tip, so this must be recomputed per row rather than reused from the whole
// shape's bounding box, or evenly-spread columns can land outside a narrower row's real footprint.
func rowSpaceBounds(spaces []Space, row int) (col_min int, col_max int, ok bool) {
	for _, s := range spaces {
		if s.Coordinate().Y != row {
			continue
		}
		col := s.Coordinate().X + floorDiv2(row)
		if !ok {
			col_min, col_max, ok = col, col, true
			continue
		}
		col_min, col_max = min(col_min, col), max(col_max, col)
	}
	return
}

// Places an evenly spaced zigzag with a random starting row and horizontal direction.
func resolveRowsPlayerStarts(ctx *mapScriptContext, starting_distance, inset, short_edge_starts int, short_edge_bias float64) ([]playerStartInfo, error) {
	spaces := ctx.allSpaces()
	row_min, row_max, _, _ := rectangleSpaceBounds(spaces)
	half := max((row_max-row_min)/2, 1)
	dist := util.Clamp(starting_distance, 1, half)
	row_near := util.Clamp(-dist, row_min+inset, row_max-inset)
	row_far := util.Clamp(dist, row_min+inset, row_max-inset)
	n := ctx.num_players
	long_count := n - min(short_edge_starts, n)
	first_row := ctx.rng.Intn(2)
	reverse := ctx.rng.Intn(2) == 1
	starts := make([]playerStartInfo, n)
	for i := range starts {
		row, direction := row_near, game_utils.Coordinate2D{X: 0, Y: 1}
		if (i+first_row)%2 == 1 {
			row, direction = row_far, game_utils.Coordinate2D{X: 0, Y: -1}
		}
		if i >= long_count {
			row = int(math.Round(float64(row_near+row_far)/2 + short_edge_bias*float64((long_count%2)*(1-2*first_row))))
		}
		col_min, col_max, ok := rowSpaceBounds(spaces, row)
		if !ok {
			return nil, fmt.Errorf("no spaces at row %d for player starts", row)
		}
		col_min, col_max = col_min+inset, col_max-inset
		col := rowPlayerStartColumn(i, long_count, col_min, col_max)
		if i >= long_count {
			col = rowPlayerStartColumn(i-long_count, n-long_count, col_min, col_max)
		}
		if reverse {
			col = col_max - (col - col_min)
		}
		if i >= long_count {
			direction = game_utils.Coordinate2D{X: 1}
			if col == col_max {
				direction.X = -1
			}
		}
		q := col - floorDiv2(row)
		space := ctx.board.Space(game_utils.Coordinate2D{X: q, Y: row})
		if space == nil {
			return nil, fmt.Errorf("player start space is nil")
		}
		starts[i] = playerStartInfo{space: space, direction: direction}
	}
	return starts, nil
}

var polygonArms = [7][]int{
	{},
	{0},
	{0, 3},
	{0, 2, 4},
	{0, 1, 3, 4},
	{0, 1, 2, 3, 4},
	{0, 1, 2, 3, 4, 5},
}

func findArmLandSpace(ctx *mapScriptContext, dir game_utils.Coordinate2D, targetDist int, used map[uint]bool) (Space, error) {
	maxR := int(ctx.board_size)
	for delta := 0; delta <= maxR; delta++ {
		for _, sign := range []int{1, -1} {
			if delta == 0 && sign == -1 {
				continue
			}
			d := targetDist + delta*sign
			if d < 1 || d > maxR {
				continue
			}
			coord := game_utils.Coordinate2D{X: d * dir.X, Y: d * dir.Y}
			if sp := ctx.board.Space(coord); sp != nil && ctx.startAllowed(sp) && !isWaterSpace(ctx, sp) && !used[sp.Key()] {
				return sp, nil
			}
		}
	}
	for _, s := range ctx.allSpaces() {
		if ctx.startAllowed(s) && !isWaterSpace(ctx, s) && !used[s.Key()] {
			return s, nil
		}
	}
	for d := 1; d <= maxR; d++ {
		coord := game_utils.Coordinate2D{X: d * dir.X, Y: d * dir.Y}
		if sp := ctx.board.Space(coord); sp != nil && !used[sp.Key()] {
			return sp, nil
		}
	}
	for _, s := range ctx.allSpaces() {
		if !used[s.Key()] {
			return s, nil
		}
	}
	return nil, fmt.Errorf("no unused space on board for player start")
}

func resolveAsteriskPlayerStarts(ctx *mapScriptContext, outerDist, innerDist int) ([]playerStartInfo, error) {
	if ctx.num_players < 1 || ctx.num_players > 12 {
		return nil, fmt.Errorf("asterisk pattern supports 1 to 12 players, got %d", ctx.num_players)
	}
	if outerDist < 1 {
		return nil, fmt.Errorf("asterisk player starts require positive outer distance")
	}
	if innerDist <= 0 {
		innerDist = max(2, outerDist/2)
	}
	if innerDist >= outerDist {
		innerDist = outerDist - 1
	}
	offset := ctx.rng.Intn(6)
	type armSlot struct{ arm, dist int }
	var slots []armSlot
	for _, a := range polygonArms[min(ctx.num_players, 6)] {
		slots = append(slots, armSlot{(a + offset) % 6, outerDist})
	}
	if ctx.num_players > 6 {
		for _, a := range polygonArms[ctx.num_players-6] {
			slots = append(slots, armSlot{(a + offset) % 6, innerDist})
		}
	}
	dirs := game_utils.AxialDirectionVectors()
	starts := make([]playerStartInfo, len(slots))
	usedKeys := make(map[uint]bool)
	for i, slot := range slots {
		dir := dirs[slot.arm]
		sp, err := findArmLandSpace(ctx, dir, slot.dist, usedKeys)
		if err != nil {
			return nil, err
		}
		usedKeys[sp.Key()] = true
		starts[i] = playerStartInfo{space: sp, direction: dir}
	}
	return starts, nil
}
