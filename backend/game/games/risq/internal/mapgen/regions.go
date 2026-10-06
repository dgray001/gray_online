package mapgen

import (
	"encoding/json"
	"fmt"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

type regionsSevenParams struct {
	Names       []string `json:"names,omitempty"`
	CenterBonus *float64 `json:"center_bonus,omitempty"`
	OuterBonus  *float64 `json:"outer_bonus,omitempty"`
}

func stepRegionsSeven(ctx *mapScriptContext, raw json.RawMessage) error {
	if ctx.shape != "hexagon" && ctx.shape != "ring" {
		return fmt.Errorf("regions_seven: only valid on hexagon or ring shapes, got %q", ctx.shape)
	}
	p, err := decodeStepParams[regionsSevenParams](raw, "regions_seven")
	if err != nil {
		return err
	}

	centerBonus := float64(50)
	if p.CenterBonus != nil {
		centerBonus = *p.CenterBonus
	}
	outerBonus := float64(30)
	if p.OuterBonus != nil {
		outerBonus = *p.OuterBonus
	}

	names := append([]string{}, p.Names...)
	if len(names) < 7 {
		names = append(names, defs.RandomRegionNames(ctx.rng, 7-len(names), names)...)
	}
	if len(names) < 7 {
		return fmt.Errorf("regions_seven: not enough region names available")
	}
	n := int(ctx.board_size)
	total := len(ctx.allSpaces())
	center_radius := 0
	for 7*hexAreaForSize(center_radius) < total {
		center_radius++
	}
	center_keys := make(map[uint]bool)
	for _, space := range ctx.allSpaces() {
		if int(game_utils.AxialDistance(game_utils.Coordinate2D{}, space.Coordinate())) <= center_radius {
			center_keys[space.Key()] = true
		}
	}
	if len(center_keys) > 0 {
		if err := ctx.board.AddRegion(names[0], centerBonus, center_keys); err != nil {
			return err
		}
	}
	sector_keys := [6]map[uint]bool{}
	for i := range sector_keys {
		sector_keys[i] = make(map[uint]bool)
	}
	for d := center_radius + 1; d <= n; d++ {
		ring := hexRingSectors(game_utils.Coordinate2D{}, d)
		for i, coords := range ring {
			for _, c := range coords {
				if space := ctx.board.Space(c); space != nil {
					sector_keys[i][space.Key()] = true
				}
			}
		}
	}
	for i := 0; i < 6; i++ {
		if len(sector_keys[i]) > 0 {
			if err := ctx.board.AddRegion(names[i+1], outerBonus, sector_keys[i]); err != nil {
				return err
			}
		}
	}
	return nil
}
