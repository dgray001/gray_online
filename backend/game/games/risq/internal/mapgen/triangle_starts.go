package mapgen

import (
	"fmt"
	"math"

	"github.com/dgray001/gray_online/game/game_utils"
)

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
