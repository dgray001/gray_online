package mapgen

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
)

type shapeParams struct {
	Kind      string     `json:"kind"`
	Size      ScriptExpr `json:"size,omitempty"`
	InnerSize ScriptExpr `json:"inner_size,omitempty"`
	Thickness ScriptExpr `json:"thickness,omitempty"`
	Rows      ScriptExpr `json:"rows,omitempty"`
	Cols      ScriptExpr `json:"cols,omitempty"`
}

func stepShape(ctx *mapScriptContext, raw json.RawMessage) error {
	if ctx.shape != "" {
		return fmt.Errorf("shape: already declared as %q", ctx.shape)
	}
	p, err := decodeStepParams[shapeParams](raw, "shape")
	if err != nil {
		return err
	}
	ctx.vars["recommended"] = float64(recommendedShapeSize(p.Kind, ctx.num_players))
	switch p.Kind {
	case "hexagon":
		size, err := p.Size.resolveInt(ctx.vars)
		if err != nil {
			return err
		}
		if size < 0 {
			return fmt.Errorf("shape: hexagon size must be >= 0")
		}
		ctx.board.Allocate(uint16(size))
		ctx.shape = "hexagon"
	case "rectangle":
		rows, err := p.Rows.resolveInt(ctx.vars)
		if err != nil {
			return err
		}
		cols, err := p.Cols.resolveInt(ctx.vars)
		if err != nil {
			return err
		}
		if rows < 1 || cols < 1 {
			return fmt.Errorf("shape: rectangle rows and cols must be >= 1")
		}
		ctx.board.Allocate(rectangleRequiredBoardSize(rows, cols))
		row_min := -((rows - 1) / 2)
		row_max := row_min + rows - 1
		col_min := -((cols - 1) / 2)
		col_max := col_min + cols - 1
		kept := 0
		for _, space := range ctx.allSpaces() {
			row := space.Coordinate().Y
			col := space.Coordinate().X + floorDiv2(row)
			if row < row_min || row > row_max || col < col_min || col > col_max {
				ctx.board.RemoveSpace(space)
				continue
			}
			kept++
		}
		if kept < rows*cols {
			return fmt.Errorf("shape: board_size too small to fit a %dx%d rectangle (increase board_size)", cols, rows)
		}
		ctx.shape = "rectangle"
	case "ring":
		size, err := p.Size.resolveInt(ctx.vars)
		if err != nil {
			return err
		}
		var inner_size int
		if p.Thickness.provided() {
			thickness, err := p.Thickness.resolveInt(ctx.vars)
			if err != nil {
				return err
			}
			inner_size = size - thickness
		} else {
			inner_size, err = p.InnerSize.resolveInt(ctx.vars)
			if err != nil {
				return err
			}
		}
		if size < 0 || inner_size < 0 || inner_size >= size {
			return fmt.Errorf("shape: ring requires 0 <= inner_size < size")
		}
		ctx.board.Allocate(uint16(size))
		for _, space := range ctx.allSpaces() {
			d := int(game_utils.AxialDistance(game_utils.Coordinate2D{}, space.Coordinate()))
			if d > size || d <= inner_size {
				ctx.board.RemoveSpace(space)
			}
		}
		ctx.shape = "ring"
	case "triangle":
		size, err := p.Size.resolveInt(ctx.vars)
		if err != nil {
			return err
		}
		if size < 0 {
			return fmt.Errorf("shape: triangle size must be >= 0")
		}
		offset := triangleCenterOffset(size)
		ctx.board.Allocate(triangleRequiredBoardSize(size))
		for _, space := range ctx.allSpaces() {
			q, r := space.Coordinate().X, space.Coordinate().Y
			if q < -offset || r < -offset || q+r > size-2*offset {
				ctx.board.RemoveSpace(space)
			}
		}
		ctx.shape = "triangle"
	default:
		return fmt.Errorf("shape: unknown kind %q", p.Kind)
	}
	ctx.board_size = ctx.board.BoardSize()
	ctx.vars["board_size"] = float64(ctx.board.BoardSize())
	return nil
}

func ringArea(outer int) int {
	inner := int(math.Round(float64(outer) / 2))
	return hexAreaForSize(outer) - hexAreaForSize(inner)
}

func recommendedShapeSize(kind string, num_players int) int {
	n := int(recommendedBoardSize(num_players))
	hex_area := hexAreaForSize(n)
	switch kind {
	case "rectangle":
		return int(math.Round(math.Sqrt(float64(hex_area))))
	case "triangle":
		return int(math.Round((-3 + math.Sqrt(1+8*float64(hex_area))) / 2))
	case "ring":
		outer := n
		for ringArea(outer) < hex_area {
			outer++
		}
		if outer > n && util.AbsInt(ringArea(outer-1)-hex_area) <= util.AbsInt(ringArea(outer)-hex_area) {
			outer--
		}
		return outer
	default:
		return n
	}
}

func triangleCenterOffset(size int) int {
	return int(math.Round(float64(size) / 3))
}

func triangleRequiredBoardSize(size int) uint16 {
	offset := triangleCenterOffset(size)
	required := max(2*offset, size-offset)
	if required < 0 {
		required = 0
	}
	return uint16(required)
}

// Computes the exact hex board radius needed to fit an R x C rectangle, checking the true axial
// distance of each row's two extreme columns rather than approximating with a closed-form formula
// (which broke for asymmetric row/col spans, e.g. even cols) -- must match stepShape's own carving.
func rectangleRequiredBoardSize(rows int, cols int) uint16 {
	row_min := -((rows - 1) / 2)
	row_max := row_min + rows - 1
	col_min := -((cols - 1) / 2)
	col_max := col_min + cols - 1
	required := 0
	for row := row_min; row <= row_max; row++ {
		shift := floorDiv2(row)
		for _, col := range [2]int{col_min, col_max} {
			x := col - shift
			d := int(game_utils.AxialDistance(game_utils.Coordinate2D{}, game_utils.Coordinate2D{X: x, Y: row}))
			required = max(required, d)
		}
	}
	return uint16(required)
}
