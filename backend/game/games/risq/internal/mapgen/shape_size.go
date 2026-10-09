package mapgen

import (
	"fmt"
	"math"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func defaultShapeSize(kind string, map_size defs.MapSize) (map[string]int, error) {
	radius := [...]int{4, 5, 6, 7, 7, 8, 8, 9, 10}[map_size-1]
	area := float64(hexAreaForSize(radius))
	switch kind {
	case "hexagon":
		return map[string]int{"radius": radius}, nil
	case "ring":
		outer := [...]int{4, 5, 5, 6, 7, 7, 8, 9, 10}[map_size-1]
		return map[string]int{"outer_radius": outer, "inner_radius": outer / 2}, nil
	case "rectangle":
		dimensions := [...][2]int{{6, 9}, {8, 12}, {9, 14}, {10, 15}, {11, 16}, {12, 18}, {13, 19}, {14, 21}, {15, 23}}[map_size-1]
		return map[string]int{"rows": dimensions[0], "cols": dimensions[1]}, nil
	case "triangle":
		return map[string]int{"edge_length": int(math.Round((-3 + math.Sqrt(1+8*area)) / 2))}, nil
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
