package testconfig

import (
	"path/filepath"
	"runtime"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func Dir() string {
	_, source, _, _ := runtime.Caller(0)
	return filepath.Dir(source)
}

func Load() error {
	return defs.LoadConfig(Dir())
}
