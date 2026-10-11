package risq

import (
	"errors"
	"math"
)

func resolvePopulationLimit(raw any) (uint16, error) {
	if raw == nil {
		return 100, nil
	}
	value, ok := raw.(float64)
	if !ok || math.IsNaN(value) || value < 25 || value > 200 || value != math.Trunc(value) {
		return 0, errors.New("population_limit must be an integer between 25 and 200")
	}
	return uint16(value), nil
}
