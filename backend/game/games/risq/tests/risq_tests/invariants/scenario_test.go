package invariants

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

type scenario struct {
	players     int
	turns       int
	capLossTurn int
	setup       func(*testing.T) *harness.Game
	orders      func(*harness.Game, int, int) []defs.OrderFromFrontend
	verify      func(*harness.Game, int)
}

func noConfig() {}
