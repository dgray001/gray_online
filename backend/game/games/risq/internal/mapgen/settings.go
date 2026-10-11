package mapgen

import "github.com/dgray001/gray_online/game/games/risq/internal/defs"

type Settings struct {
	MapSize             defs.MapSize
	StartingStockpile   defs.StartingStockpile
	MapResources        defs.MapResourceLevel
	StartingDevelopment defs.StartingDevelopment
	StartBalance        defs.StartBalance
}

func DefaultStartingBank(stockpile defs.StartingStockpile) StartingBank {
	switch stockpile {
	case defs.StartingStockpile_LOW:
		return StartingBank{Food: 100, Wood: 100, Stone: 50}
	case defs.StartingStockpile_HIGH:
		return StartingBank{Food: 500, Wood: 500, Stone: 250, Gold: 100}
	case defs.StartingStockpile_DEATHMATCH:
		return StartingBank{Food: 10000, Wood: 10000, Stone: 5000, Gold: 5000}
	default:
		return StartingBank{Food: 200, Wood: 200, Stone: 100}
	}
}

func ResolveSettings(raw map[string]any, num_players int) Settings {
	return Settings{
		MapSize:             defs.ResolveMapSize(raw["map_size"], num_players),
		StartingStockpile:   defs.ResolveSetupSetting(raw["starting_stockpile"], defs.StartingStockpile_STANDARD, defs.StartingStockpile_DEATHMATCH),
		MapResources:        defs.ResolveSetupSetting(raw["map_resources"], defs.MapResourceLevel_STANDARD, defs.MapResourceLevel_ABUNDANT),
		StartingDevelopment: defs.ResolveSetupSetting(raw["starting_development"], defs.StartingDevelopment_STANDARD, defs.StartingDevelopment_ADVANCED),
		StartBalance:        defs.ResolveSetupSetting(raw["start_balance"], defs.StartBalance_MIRRORED, defs.StartBalance_MIRRORED),
	}
}
