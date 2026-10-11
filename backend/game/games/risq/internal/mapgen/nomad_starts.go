package mapgen

import (
	"fmt"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
	"math/rand"
)

type nomadPlacer struct {
	rng         *rand.Rand
	zones       []Zone
	used_spaces map[uint]bool
	used_zones  map[uint]bool
	own_zones   map[int][]Zone
	next_free   int
}

func (n *nomadPlacer) takeFree() Zone {
	for ; n.next_free < len(n.zones); n.next_free++ {
		if z := n.zones[n.next_free]; !z.Occupied() {
			n.next_free++
			return z
		}
	}
	return nil
}

// Buildings need an unoccupied zone; units may share zones with resources, buildings and each other
func (n *nomadPlacer) pick(player int, building bool) Zone {
	fits := func(z Zone) bool { return !building || !z.Occupied() }
	tiers := []func(Zone) bool{
		func(z Zone) bool { return fits(z) && !n.used_spaces[z.Space().Key()] },
		func(z Zone) bool { return fits(z) && !n.used_zones[z.Key()] },
	}
	for _, tier := range tiers {
		for _, z := range n.zones {
			if tier(z) {
				return z
			}
		}
	}
	if own := n.own_zones[player]; !building && len(own) > 0 {
		return own[n.rng.Intn(len(own))]
	}
	for _, z := range n.zones {
		if fits(z) {
			return z
		}
	}
	return nil
}

func (n *nomadPlacer) place(player int, building bool) (Zone, error) {
	z := n.pick(player, building)
	if z == nil {
		return nil, fmt.Errorf("no free passable zone left for nomad starts")
	}
	n.used_spaces[z.Space().Key()], n.used_zones[z.Key()] = true, true
	n.own_zones[player] = append(n.own_zones[player], z)
	return z, nil
}

func placeNomadStarts(ctx *mapScriptContext, p playerStartsParams) error {
	zones := make([]Zone, 0)
	for _, zone := range ctx.allZones() {
		if !zone.Space().Impassable() {
			zones = append(zones, zone)
		}
	}
	placer := &nomadPlacer{rng: ctx.rng, zones: util.ShuffleFrom(ctx.rng, zones), used_spaces: map[uint]bool{}, used_zones: map[uint]bool{}, own_zones: map[int][]Zone{}}
	for player := 0; player < ctx.num_players; player++ {
		if err := placeNomadAssets(ctx, p, player, placer); err != nil {
			return err
		}
	}
	for player := 0; player < ctx.num_players; player++ {
		if err := placeNomadResources(ctx, p, placer); err != nil {
			return err
		}
		ctx.board.SetStartingBank(player, nomadStartingBank(ctx, p))
	}
	ctx.nomad_starts = true
	return nil
}

const villageCenterId uint32 = 1

// An explicit script bank is used as given; otherwise the default stockpile plus the cost of the village center nomads must build
func nomadStartingBank(ctx *mapScriptContext, p playerStartsParams) StartingBank {
	if p.StartingBank != nil {
		return *p.StartingBank
	}
	bank := DefaultStartingBank(defs.StartingStockpile(ctx.vars["starting_stockpile"]))
	for _, building := range p.Buildings {
		if building.BuildingId == villageCenterId {
			return bank
		}
	}
	cost := defs.BuildingConfigs[villageCenterId].Cost
	bank.Food, bank.Wood, bank.Stone, bank.Gold = bank.Food+cost.Food, bank.Wood+cost.Wood, bank.Stone+cost.Stone, bank.Gold+cost.Gold
	return bank
}

func placeNomadAssets(ctx *mapScriptContext, p playerStartsParams, player int, placer *nomadPlacer) error {
	for _, building := range p.Buildings {
		zone, err := placer.place(player, true)
		if err != nil {
			return err
		}
		if !ctx.board.PlaceBuilding(zone, building.BuildingId, player) {
			return fmt.Errorf("could not place nomad building %d", building.BuildingId)
		}
		if building.TerrainOverride != 0 {
			zone.SetTerrainOverride(building.TerrainOverride)
		}
	}
	for _, unit := range p.Units {
		count, err := unit.Count.resolveInt(ctx.vars)
		if err != nil {
			return err
		}
		for range count {
			zone, err := placer.place(player, false)
			if err != nil {
				return err
			}
			ctx.board.PlaceUnit(zone, unit.UnitId, player)
		}
	}
	return nil
}

func placeNomadResources(ctx *mapScriptContext, p playerStartsParams, placer *nomadPlacer) error {
	slots, err := planStartResources(ctx, p.Resources, 0)
	if err != nil {
		return err
	}
	for _, slot := range slots {
		if zone := placer.takeFree(); zone != nil {
			ctx.board.PlaceResource(zone, slot.resource_id)
		}
	}
	for _, override := range p.ZoneTerrainOverrides {
		if zone := placer.takeFree(); zone != nil {
			zone.SetTerrainOverride(override.TerrainId)
		}
	}
	return nil
}
