package mapgen

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/dgray001/gray_online/util"
)

// A blob size given as one number or expression, or as [min, max] drawn uniformly for each blob
type BlobSize struct {
	bounds []ScriptExpr
}

func (s *BlobSize) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '[' {
		if err := json.Unmarshal(data, &s.bounds); err != nil {
			return err
		}
		if len(s.bounds) != 2 {
			return fmt.Errorf("blob size range needs exactly [min, max], got %d values", len(s.bounds))
		}
		return nil
	}
	s.bounds = make([]ScriptExpr, 1)
	return json.Unmarshal(data, &s.bounds[0])
}

// The inclusive bounds; a single size has equal bounds and an unset size is 0
func (s BlobSize) resolve(vars map[string]float64) (int, int, error) {
	if len(s.bounds) == 0 {
		return 0, 0, nil
	}
	low, err := s.bounds[0].resolveInt(vars)
	if err != nil || len(s.bounds) == 1 {
		return low, low, err
	}
	high, err := s.bounds[1].resolveInt(vars)
	return low, high, err
}

// A fixed size draws nothing from the rng, so maps without ranges stay reproducible
func pickBlobSize(rng *rand.Rand, low int, high int) int {
	if low == high {
		return low
	}
	return util.RandomIntFrom(rng, low, high)
}
