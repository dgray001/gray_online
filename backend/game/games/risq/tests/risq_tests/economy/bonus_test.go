package economy

import (
	"math"
	"testing"
)

func TestTargetedBonusOnlyAffectsMatchingOpponent(t *testing.T) {
	for name, building := range map[string]bool{"building": true, "unit": false} {
		t.Run(name, func(t *testing.T) {
			without := bonusDamage(t, building, false)
			with := bonusDamage(t, building, true)
			want := 0.0
			if building {
				want = 6
			}
			if math.Abs((with-without)-want) > 0.00001 {
				t.Errorf("bonus damage %v, want %v (with %v, without %v)", with-without, want, with, without)
			}
		})
	}
}
