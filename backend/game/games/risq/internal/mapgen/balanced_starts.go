package mapgen

import "github.com/dgray001/gray_online/game/games/risq/internal/defs"

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
