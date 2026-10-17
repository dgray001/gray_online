package mapgen

import (
	"math"
)

type distanceToJSON struct {
	Target   terrainFilterJSON `json:"target"`
	Region   string            `json:"region,omitempty"`
	Distance ScriptExpr        `json:"distance,omitempty"`
	Min      ScriptExpr        `json:"min,omitempty"`
	Max      ScriptExpr        `json:"max,omitempty"`
}

func (d distanceToJSON) empty() bool {
	return d.Target.empty() && d.Region == ""
}

func (d distanceToJSON) resolveRange(ctx *mapScriptContext) (int, int, error) {
	if d.Distance.provided() {
		val, err := d.Distance.resolveInt(ctx.vars)
		return val, val, err
	}
	minVal, maxVal := 1, math.MaxInt/2
	if d.Min.provided() {
		var err error
		if minVal, err = d.Min.resolveInt(ctx.vars); err != nil {
			return 0, 0, err
		}
	}
	if d.Max.provided() {
		var err error
		if maxVal, err = d.Max.resolveInt(ctx.vars); err != nil {
			return 0, 0, err
		}
	}
	return minVal, maxVal, nil
}

func (d distanceToJSON) matcher(ctx *mapScriptContext) (func(Space) bool, error) {
	minDist, maxDist, err := d.resolveRange(ctx)
	if err != nil {
		return nil, err
	}
	targetMatcher, err := d.Target.matcher(ctx)
	if err != nil {
		return nil, err
	}
	region := ctx.region(d.Region)
	var starts []Space
	for _, s := range ctx.allSpaces() {
		if (region == nil || region[s.Key()]) && (d.Target.empty() || targetMatcher(s)) {
			starts = append(starts, s)
		}
	}
	if len(starts) == 0 {
		return func(Space) bool { return false }, nil
	}
	limit := maxDist
	if limit >= len(ctx.allSpaces()) {
		limit = minDist
	}
	dist := bfsDistances(starts, limit,
		func(s Space) uint { return s.Key() },
		func(s Space) []Space { return s.SortedAdjacent() },
	)
	return func(s Space) bool {
		dVal, ok := dist[s.Key()]
		if !ok {
			dVal = math.MaxInt / 2
		}
		return dVal >= minDist && dVal <= maxDist
	}, nil
}
