package risq

import "github.com/dgray001/gray_online/game/games/risq/internal/defs"

const (
	earlyGameTurns          = 20
	villageCenterBuildingId = 1
)

type staminaUsage struct {
	granted int
	wasted  int
}

func (s staminaUsage) idlePct() float64 {
	if s.granted == 0 {
		return 0
	}
	return 100 * float64(s.wasted) / float64(s.granted)
}

type economyStats struct {
	villagers            staminaUsage
	villagers_early      staminaUsage
	village_center       staminaUsage
	village_center_early staminaUsage
	villagers_lost       uint
}

// Stamina the upcoming refresh will throw away because the carried-over amount would exceed the cap
func refreshWaste(o *orderableBase) int {
	return max(0, o.current_stamina+o.turn_stamina-maxStaminaFor(o.turn_stamina))
}

func addStamina(total *staminaUsage, early *staminaUsage, o *orderableBase, is_early bool) {
	for _, usage := range []*staminaUsage{total, early} {
		if usage == early && !is_early {
			continue
		}
		usage.granted += o.turn_stamina
		usage.wasted += refreshWaste(o)
	}
}

// Must run right before the per-turn stamina refresh; idle time is stamina lost to the cap, not carryover
func (r *GameRisq) recordEconomyStamina() {
	is_early := r.turn_number <= earlyGameTurns
	for _, p := range r.players {
		for _, u := range p.units {
			if !u.deleted && u.unitType() == defs.UnitType_ECONOMIC {
				addStamina(&p.economy.villagers, &p.economy.villagers_early, &u.orderableBase, is_early)
			}
		}
		for _, b := range p.buildings {
			if !b.deleted && b.building_id == villageCenterBuildingId && !b.underConstruction() {
				addStamina(&p.economy.village_center, &p.economy.village_center_early, &b.orderableBase, is_early)
			}
		}
	}
}
