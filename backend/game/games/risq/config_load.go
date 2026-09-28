package risq

import "github.com/dgray001/gray_online/game/games/risq/internal/defs"

// Exposes defs.LoadConfig to binaries outside risq, which can't import internal packages
func LoadConfig(dir string) error {
	return defs.LoadConfig(dir)
}
