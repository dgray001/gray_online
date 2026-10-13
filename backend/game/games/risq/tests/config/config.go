package testconfig

import (
	"path/filepath"
	"runtime"
	"runtime/debug"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

// Test packages allocate heavily on many cores, where default GC pacing dominates their runtime
func init() {
	debug.SetGCPercent(800)
}

func Dir() string {
	_, source, _, _ := runtime.Caller(0)
	return filepath.Dir(source)
}

func Load() error {
	return defs.LoadConfig(Dir())
}
