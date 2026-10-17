package mapgen

import (
	"encoding/json"
	"fmt"

	"github.com/dgray001/gray_online/game/game_utils"
)

type rotateParams struct {
	Count  ScriptExpr        `json:"count,omitempty"`
	Steps  ScriptExpr        `json:"steps,omitempty"`
	On     terrainFilterJSON `json:"on,omitempty"`
	Region string            `json:"region,omitempty"`
}

func resolveRotSteps(ctx *mapScriptContext, p rotateParams) ([]int, error) {
	steps, _ := p.Steps.resolveInt(ctx.vars)
	if steps != 0 {
		return []int{steps}, nil
	}
	count, _ := p.Count.resolveInt(ctx.vars)
	if count > 1 && 6%count == 0 {
		var rotSteps []int
		for s := 1; s < count; s++ {
			rotSteps = append(rotSteps, s*(6/count))
		}
		return rotSteps, nil
	}
	return nil, fmt.Errorf("rotate: count must be 2, 3, or 6 (or steps must be non-zero)")
}

func stepRotate(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[rotateParams](raw, "rotate")
	if err != nil {
		return err
	}
	rotSteps, err := resolveRotSteps(ctx, p)
	if err != nil {
		return err
	}
	if p.Region == "" && p.On.empty() {
		return fmt.Errorf("rotate: requires 'on' or 'region'")
	}
	region := ctx.region(p.Region)
	matches, err := p.On.matcher(ctx)
	if err != nil {
		return err
	}
	type change struct {
		coord game_utils.Coordinate2D
		tid   uint32
	}
	var changes []change
	for _, space := range ctx.allSpaces() {
		if (region == nil || region[space.Key()]) && matches(space) {
			for _, step := range rotSteps {
				changes = append(changes, change{rotateAxial(space.Coordinate(), step), space.Terrain()})
			}
		}
	}
	for _, c := range changes {
		if s := ctx.board.Space(c.coord); s != nil {
			s.SetTerrain(c.tid)
			if region != nil {
				region[s.Key()] = true
			}
		}
	}
	return nil
}
