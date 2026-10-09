package mapgen

import (
	"encoding/json"
	"fmt"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func stepRegionsFour(ctx *mapScriptContext, raw json.RawMessage) error {
	if ctx.shape != "triangle" {
		return fmt.Errorf("regions_four: only valid on triangle shapes, got %q", ctx.shape)
	}
	p, err := decodeStepParams[regionsBonusParams](raw, "regions_four")
	if err != nil {
		return err
	}
	centerBonus, err := p.CenterBonus.resolve(ctx.vars)
	if err != nil {
		return err
	}
	outerBonus, err := p.OuterBonus.resolve(ctx.vars)
	if err != nil {
		return err
	}
	names := append([]string{}, p.Names...)
	if len(names) < 4 {
		names = append(names, defs.RandomRegionNames(ctx.rng, 4-len(names), names)...)
	}
	if len(names) < 4 {
		return fmt.Errorf("regions_four: not enough region names available")
	}
	edge := int(ctx.vars["edge_length"])
	if edge < 2 {
		return fmt.Errorf("regions_four: triangle needs edge_length of at least 2")
	}
	offset := triangleCenterOffset(edge)
	groups := [4]map[uint]bool{}
	for i := range groups {
		groups[i] = map[uint]bool{}
	}
	for _, space := range ctx.allSpaces() {
		coordinate := space.Coordinate()
		x, y := coordinate.X+offset, coordinate.Y+offset
		index := 0
		switch {
		case 2*(edge-x-y) > edge:
			index = 1
		case 2*x > edge:
			index = 2
		case 2*y > edge:
			index = 3
		}
		groups[index][space.Key()] = true
	}
	for i, keys := range groups {
		bonus := outerBonus
		if i == 0 {
			bonus = centerBonus
		}
		if err := ctx.board.AddRegion(names[i], bonus, keys); err != nil {
			return err
		}
	}
	return nil
}
