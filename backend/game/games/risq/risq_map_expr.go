package risq

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/dgray001/gray_online/util"
)

// A map script number that may be a plain JSON number or an arithmetic expression string
// referencing script variables (e.g. "2 + 2 * board_size")
type ScriptExpr struct {
	raw json.RawMessage
}

func (e *ScriptExpr) UnmarshalJSON(data []byte) error {
	e.raw = append(json.RawMessage(nil), data...)
	return nil
}

func (e ScriptExpr) resolve(vars map[string]float64) (float64, error) {
	if len(e.raw) == 0 {
		return 0, nil
	}
	var num float64
	if err := json.Unmarshal(e.raw, &num); err == nil {
		return num, nil
	}
	var str string
	if err := json.Unmarshal(e.raw, &str); err != nil {
		return 0, fmt.Errorf("expected a number or expression string, got %s", string(e.raw))
	}
	return util.EvalExpr(str, vars)
}

func (e ScriptExpr) resolveInt(vars map[string]float64) (int, error) {
	v, err := e.resolve(vars)
	if err != nil {
		return 0, err
	}
	return int(math.Round(v)), nil
}

func (e ScriptExpr) provided() bool {
	return len(e.raw) > 0
}
