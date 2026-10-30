package mapgen

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

type shapeParams struct {
	Kind string                `json:"kind"`
	Size map[string]ScriptExpr `json:"size,omitempty"`
}

func stepShape(ctx *mapScriptContext, raw json.RawMessage) error {
	if ctx.shape != "" {
		return fmt.Errorf("shape: already declared as %q", ctx.shape)
	}
	p, err := decodeStepParams[shapeParams](raw, "shape")
	if err != nil {
		return err
	}
	sizes, err := resolveShapeSize(ctx, p)
	if err != nil {
		return err
	}
	for name, value := range sizes {
		ctx.vars[name] = float64(value)
	}
	ctx.insets, ctx.zones = nil, nil
	switch p.Kind {
	case "hexagon":
		size := sizes["radius"]
		if size < 0 {
			return fmt.Errorf("shape: hexagon size must be >= 0")
		}
		ctx.board.Allocate(uint16(size))
		ctx.shape = "hexagon"
	case "rectangle":
		rows, cols := sizes["rows"], sizes["cols"]
		if rows < 1 || cols < 1 {
			return fmt.Errorf("shape: rectangle rows and cols must be >= 1")
		}
		ctx.board.Allocate(rectangleRequiredBoardSize(rows, cols))
		row_min := -(rows / 2)
		row_max := row_min + rows - 1
		col_min := -(cols / 2)
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
		size, inner_size := sizes["outer_radius"], sizes["inner_radius"]
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
		size := sizes["edge_length"]
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
	row_min := -(rows / 2)
	row_max := row_min + rows - 1
	col_min := -(cols / 2)
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

func defaultShapeSize(kind string, map_size defs.MapSize) (map[string]int, error) {
	radius := [...]int{4, 5, 5, 6, 7, 7, 8, 8, 10}[map_size-1]
	switch kind {
	case "hexagon":
		return map[string]int{"radius": radius}, nil
	case "ring":
		outer := [...]int{4, 5, 5, 6, 7, 7, 8, 9, 10}[map_size-1]
		inner := [...]int{2, 3, 2, 3, 4, 3, 4, 5, 5}[map_size-1]
		return map[string]int{"outer_radius": outer, "inner_radius": inner}, nil
	case "rectangle":
		dimensions := [...][2]int{{5, 9}, {7, 12}, {7, 14}, {9, 15}, {9, 17}, {9, 19}, {9, 21}, {11, 22}, {11, 26}}[map_size-1]
		return map[string]int{"rows": dimensions[0], "cols": dimensions[1]}, nil
	case "triangle":
		edge_length := [...]int{9, 9, 12, 12, 15, 15, 18, 18, 21}[map_size-1]
		return map[string]int{"edge_length": edge_length}, nil
	default:
		return nil, fmt.Errorf("shape: unknown kind %q", kind)
	}
}

func resolveShapeSize(ctx *mapScriptContext, p shapeParams) (map[string]int, error) {
	sizes, err := defaultShapeSize(p.Kind, defs.MapSize(ctx.vars["map_size"]))
	if err != nil {
		return nil, err
	}
	for name, expression := range p.Size {
		if _, ok := sizes[name]; !ok {
			return nil, fmt.Errorf("shape: unknown %s size field %q", p.Kind, name)
		}
		value, err := expression.resolveInt(ctx.vars)
		if err != nil {
			return nil, err
		}
		sizes[name] = value
	}
	return sizes, nil
}
