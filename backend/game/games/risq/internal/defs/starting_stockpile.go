package defs

type StartingStockpile uint8

const (
	StartingStockpile_NONE StartingStockpile = iota
	StartingStockpile_LOW
	StartingStockpile_STANDARD
	StartingStockpile_HIGH
	StartingStockpile_DEATHMATCH
)
