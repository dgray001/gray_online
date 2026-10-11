package mapgen

import (
	"fmt"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func (p playerStartsParams) startBalance(vars map[string]float64) (defs.StartBalance, error) {
	value := vars["start_balance"]
	if p.StartBalance != nil {
		resolved, err := p.StartBalance.resolve(vars)
		if err != nil {
			return 0, err
		}
		value = resolved
	}
	if value != float64(defs.StartBalance_RANDOM) && value != float64(defs.StartBalance_MIRRORED) {
		return 0, fmt.Errorf("start_balance must be Random or Mirrored")
	}
	return defs.StartBalance(value), nil
}
