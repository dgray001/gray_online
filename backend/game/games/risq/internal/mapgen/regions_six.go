package mapgen

import (
	"encoding/json"
	"fmt"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

type regionsBonusParams struct {
	Names       []string   `json:"names,omitempty"`
	CenterBonus ScriptExpr `json:"center_bonus,omitempty"`
	OuterBonus  ScriptExpr `json:"outer_bonus,omitempty"`
}

const rectangleRegionRows, rectangleRegionCols = 2, 3

func stepRegionsSix(ctx *mapScriptContext, raw json.RawMessage) error {
	if ctx.shape != "rectangle" {
		return fmt.Errorf("regions_six: only valid on rectangle shapes, got %q", ctx.shape)
	}
	p, err := decodeStepParams[regionsBonusParams](raw, "regions_six")
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
	if len(names) < 6 {
		names = append(names, defs.RandomRegionNames(ctx.rng, 6-len(names), names)...)
	}
	if len(names) < 6 {
		return fmt.Errorf("regions_six: not enough region names available")
	}
	spaces := ctx.allSpaces()
	rowMin, rowMax, colMin, colMax := rectangleSpaceBounds(spaces)
	rows, cols := rowMax-rowMin+1, colMax-colMin+1
	if rows < rectangleRegionRows || cols < rectangleRegionCols {
		return fmt.Errorf("regions_six: rectangle needs at least 2 rows and 3 columns")
	}
	groups := [rectangleRegionRows * rectangleRegionCols]map[uint]bool{}
	for i := range groups {
		groups[i] = map[uint]bool{}
	}
	for _, space := range spaces {
		coordinate := space.Coordinate()
		row := rectangleRegionRows * (coordinate.Y - rowMin) / rows
		col := rectangleRegionCols * (coordinate.X + floorDiv2(coordinate.Y) - colMin) / cols
		groups[row*rectangleRegionCols+col][space.Key()] = true
	}
	for i, keys := range groups {
		bonus := outerBonus
		if i%rectangleRegionCols == rectangleRegionCols/2 {
			bonus = centerBonus
		}
		if err := ctx.board.AddRegion(names[i], bonus, keys); err != nil {
			return err
		}
	}
	return nil
}
