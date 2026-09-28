package mapgen

import (
	"encoding/json"
	"fmt"

	"github.com/dgray001/gray_online/game/game_utils"
)

type mirrorParams struct {
	Step mapScriptStepJSON `json:"step"`
}

// Replays a wrapped step's board changes rotated to every other player start direction
func stepMirror(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[mirrorParams](raw, "mirror")
	if err != nil {
		return err
	}
	fn, ok := mapStepRegistry[p.Step.Step]
	if !ok {
		return fmt.Errorf("mirror: unknown step %q", p.Step.Step)
	}
	if len(ctx.player_starts) < 2 {
		return fn(ctx, p.Step.Params)
	}
	terrain_before := make(map[uint]uint32)
	for _, space := range ctx.allSpaces() {
		terrain_before[space.Key()] = space.Terrain()
	}
	resource_zones_before := make(map[uint]bool)
	for _, zone := range ctx.allZones() {
		if _, ok := zone.ResourceId(); ok {
			resource_zones_before[zone.Key()] = true
		}
	}
	if err := fn(ctx, p.Step.Params); err != nil {
		return err
	}
	type terrainChange struct {
		coordinate game_utils.Coordinate2D
		terrain_id uint32
	}
	terrain_changes := make([]terrainChange, 0)
	for _, space := range ctx.allSpaces() {
		if before, ok := terrain_before[space.Key()]; ok && before != space.Terrain() {
			terrain_changes = append(terrain_changes, terrainChange{coordinate: space.Coordinate(), terrain_id: space.Terrain()})
		}
	}
	type resourceChange struct {
		space_coordinate game_utils.Coordinate2D
		zone_coordinate  game_utils.Coordinate2D
		resource_id      uint32
	}
	resource_changes := make([]resourceChange, 0)
	for _, zone := range ctx.allZones() {
		if resource_id, ok := zone.ResourceId(); ok && !resource_zones_before[zone.Key()] {
			resource_changes = append(resource_changes, resourceChange{
				space_coordinate: zone.Space().Coordinate(),
				zone_coordinate:  zone.Local(),
				resource_id:      resource_id,
			})
		}
	}
	base_dir := game_utils.AxialDirectionIndex(ctx.player_starts[0].direction)
	for i := 1; i < len(ctx.player_starts); i++ {
		dir := game_utils.AxialDirectionIndex(ctx.player_starts[i].direction)
		if base_dir < 0 || dir < 0 {
			continue
		}
		k := dir - base_dir
		for _, change := range terrain_changes {
			rotated := rotateAxial(change.coordinate, k)
			if space := ctx.board.Space(rotated); space != nil {
				space.SetTerrain(change.terrain_id)
			}
		}
		for _, change := range resource_changes {
			rotated_space_c := rotateAxial(change.space_coordinate, k)
			space := ctx.board.Space(rotated_space_c)
			if space == nil {
				continue
			}
			local := change.zone_coordinate
			if local.X != 0 || local.Y != 0 {
				local = rotateAxial(local, k)
			}
			zone := space.Zone(local)
			if zone == nil || zone.Occupied() {
				continue
			}
			ctx.board.PlaceResource(zone, change.resource_id)
		}
	}
	return nil
}
