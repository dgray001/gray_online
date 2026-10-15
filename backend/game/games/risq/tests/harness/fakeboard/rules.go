package fakeboard

import (
	"fmt"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/internal/mapgen"
)

// Rejects the same inputs the engine does: a reused name, an unknown space, or a space already in another region
func (b *Board) AddRegion(name string, gold_bonus float64, keys map[uint]bool) error {
	for _, existing := range b.Regions {
		if existing.Name == name {
			return fmt.Errorf("region %q already exists", name)
		}
	}
	for key := range keys {
		if b.byKey[key] == nil {
			return fmt.Errorf("region %q: invalid space coordinate key %d", name, key)
		}
		for _, existing := range b.Regions {
			if existing.Keys[key] {
				return fmt.Errorf("region %q: space %d already belongs to region %q", name, key, existing.Name)
			}
		}
	}
	b.Regions = append(b.Regions, Region{Name: name, GoldBonus: gold_bonus, Keys: keys})
	return nil
}

func (b *Board) SetStartingBank(player_index int, bank mapgen.StartingBank) {
	b.Banks[player_index] = bank
}

func (b *Board) GrantStartingTech(player_index int, tech_id uint32) error {
	if _, ok := defs.TechConfigs[tech_id]; !ok {
		return fmt.Errorf("unknown starting tech %d", tech_id)
	}
	b.Techs[player_index] = append(b.Techs[player_index], tech_id)
	return nil
}

func (b *Board) SetUnlimitedPopulation() {
	b.Unlimited = true
}

func (b *Board) SetSpaceGoldIncome(gold float64) {
	b.GoldIncome = &gold
}

func (b *Board) SetMercenariesNeedRegion(need bool) {
	b.NeedRegion = &need
}

// The engine silently drops a resource placed on an occupied zone, so the fake records it for tests to assert never happens
func (b *Board) PlaceResource(z mapgen.Zone, resource_id uint32) {
	zone := z.(*zone)
	if zone.Occupied() {
		b.Violations = append(b.Violations, fmt.Sprintf("resource %d placed on occupied zone %d", resource_id, zone.key))
		return
	}
	zone.resource = resource_id
}

func (b *Board) PlaceBuilding(z mapgen.Zone, building_id uint32, player_index int, resources_left ...float64) bool {
	zone := z.(*zone)
	if zone.Occupied() {
		return false
	}
	zone.building = &Building{ID: building_id, Player: player_index}
	return true
}

func (b *Board) PlaceUnit(z mapgen.Zone, unit_id uint32, player_index int) {
	zone := z.(*zone)
	zone.units = append(zone.units, Unit{ID: unit_id, Player: player_index})
}

func (b *Board) ConnectSpaces(a, c mapgen.Space, direction int, to_direction int) error {
	first, second := a.(*space), c.(*space)
	b.distances = nil
	if direction < 0 || direction > 5 || to_direction < 0 || to_direction > 5 {
		return fmt.Errorf("seam directions %d and %d are not 0 to 5", direction, to_direction)
	}
	if first.borders[direction] != nil || second.borders[to_direction] != nil {
		return fmt.Errorf("seam edge between %v and %v is already connected", first.coord, second.coord)
	}
	first.borders[direction], second.borders[to_direction] = second, first
	first.arrivals[direction], second.arrivals[to_direction] = to_direction, direction
	first.sorted, second.sorted = nil, nil
	return nil
}

// The distance the engine reports between spaces with no path
const Unreachable = 1 << 15

// Steps between two spaces over borders, seams and links; Unreachable when there is no path
func (b *Board) Distance(a, c mapgen.Space) int {
	from, to := a.(*space), c.(*space)
	if from == to {
		return 0
	}
	if b.distances == nil {
		b.distances = map[*space][]uint16{}
		for i, s := range b.order {
			s.distanceIndex = i
		}
	}
	if b.distances[from] == nil {
		b.distances[from] = b.hopDistances(from)
	}
	return int(b.distances[from][to.distanceIndex])
}
