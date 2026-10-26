package mapgen

import (
	"encoding/json"
	"fmt"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

type regionsSevenParams struct {
	Names       []string    `json:"names,omitempty"`
	CenterBonus *ScriptExpr `json:"center_bonus,omitempty"`
	OuterBonus  *ScriptExpr `json:"outer_bonus,omitempty"`
	Rotate30    bool        `json:"rotate_30,omitempty"`

	CenterSpaceGold *ScriptExpr `json:"center_space_gold,omitempty"`
	OuterSpaceGold  *ScriptExpr `json:"outer_space_gold,omitempty"`
}

func setRegionSpaceGold(ctx *mapScriptContext, keys map[uint]bool, gold *ScriptExpr) error {
	if gold == nil {
		return nil
	}
	value, err := gold.resolve(ctx.vars)
	if err != nil {
		return err
	}
	for _, space := range ctx.allSpaces() {
		if keys[space.Key()] {
			space.SetGoldIncome(value)
		}
	}
	return nil
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
		centerBonus, err = p.CenterBonus.resolve(ctx.vars)
		if err != nil {
			return err
		}
	}
	outerBonus := float64(30)
	if p.OuterBonus != nil {
		outerBonus, err = p.OuterBonus.resolve(ctx.vars)
		if err != nil {
			return err
		}
	}

	names := append([]string{}, p.Names...)
	if len(names) < 7 {
		names = append(names, ctx.config().RandomRegionNames(ctx.rng, 7-len(names), names)...)
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
		if int(game_utils.AxialDistance(game_utils.Coordinate2D{}, space.Coordinate())) <= center_radius && !isWaterSpace(ctx, space) {
			center_keys[space.Key()] = true
		}
	}
	if len(center_keys) > 0 {
		if err := ctx.board.AddRegion(names[0], centerBonus, center_keys); err != nil {
			return err
		}
		if err := setRegionSpaceGold(ctx, center_keys, p.CenterSpaceGold); err != nil {
			return err
		}
	}
	sector_keys := [6]map[uint]bool{}
	for i := range sector_keys {
		sector_keys[i] = make(map[uint]bool)
	}
	for d := center_radius + 1; d <= n; d++ {
		ring := hexRingSectors(game_utils.Coordinate2D{}, d)
		var ringCoords []game_utils.Coordinate2D
		for i := 0; i < 6; i++ {
			ringCoords = append(ringCoords, ring[i]...)
		}
		totalLen := len(ringCoords)
		for i := 0; i < 6; i++ {
			for j := 0; j < d; j++ {
				idx := i*d + j
				if p.Rotate30 {
					idx = (i*d - d/2 + j + totalLen) % totalLen
				}
				if space := ctx.board.Space(ringCoords[idx]); space != nil && !isWaterSpace(ctx, space) {
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
			if err := setRegionSpaceGold(ctx, sector_keys[i], p.OuterSpaceGold); err != nil {
				return err
			}
		}
	}
	return nil
}

func isWaterSpace(ctx *mapScriptContext, space Space) bool {
	if space == nil {
		return false
	}
	if t, ok := ctx.config().Terrains[space.Terrain()]; ok {
		return t.Terrain_type == defs.TerrainType_WATER || t.Terrain_type == defs.TerrainType_DEEP_WATER
	}
	return space.Impassable() || space.Terrain() >= 251
}
