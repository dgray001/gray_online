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
