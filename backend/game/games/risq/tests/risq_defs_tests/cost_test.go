package risq_defs_tests

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"testing"
)

func TestCostComputePoints(t *testing.T) {
	c := defs.RisqResourceCost{Food: 10, Wood: 20, Stone: 30, Gold: 40}
	if pts := c.Points(); pts != 140 {
		t.Errorf("Expected 140 points, got %d", pts)
	}
	scaled := c.Scale(2.0)
	if scaled.Food != 20 || scaled.Gold != 80 {
		t.Errorf("Scale failed: %+v", scaled)
	}
}
