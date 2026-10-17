package mapgen

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
)

type blobPlacementJSON struct {
	Pattern  string        `json:"pattern"`
	Distance [2]ScriptExpr `json:"distance"`
	// ring only: "center" (default) measures hops from the shape's center, "edge" steps in from its boundary
	From        string             `json:"from"`
	Points      []axialParamJSON   `json:"points"`
	SeedRegion  string             `json:"seed_region,omitempty"`
	Between     json.RawMessage    `json:"between,omitempty"`
	PathCosts   map[string]float64 `json:"path_costs,omitempty"`
	FilterSeeds bool               `json:"filter_seeds,omitempty"`
	// 0 places each seed uniformly at random; 1 always takes the candidate farthest from the earlier seeds
	Evenness ScriptExpr `json:"evenness"`
	// ring only: blobs may not grow outside the distance band
	KeepInside bool `json:"keep_inside"`
}

// Spaces seeds may start on: any, or those within a [min, max] distance of the center
func blobSeedCandidates(ctx *mapScriptContext, p blobPlacementJSON) ([]Space, error) {
	if p.KeepInside && p.Pattern != "ring" {
		return nil, fmt.Errorf("blob placement keep_inside needs the ring pattern")
	}
	switch p.Pattern {
	case "", "random", "points", "line", "rectangle":
		return ctx.allSpaces(), nil
	case "ring":
		lo, err := p.Distance[0].resolveInt(ctx.vars)
		if err != nil {
			return nil, err
		}
		hi, err := p.Distance[1].resolveInt(ctx.vars)
		if err != nil {
			return nil, err
		}
		distances, err := ringDistances(ctx, p.From)
		if err != nil {
			return nil, err
		}
		candidates := make([]Space, 0)
		for _, space := range ctx.allSpaces() {
			if d := distances[space.Key()]; d >= lo && d <= hi {
				candidates = append(candidates, space)
			}
		}
		return candidates, nil
	}
	return nil, fmt.Errorf("unknown blob placement pattern %q", p.Pattern)
}

const blobSeedAttemptsPerSpacing = 10

func blobSeedFits(ctx *mapScriptContext, candidate Space, seeds []Space, spacing int) bool {
	for _, seed := range seeds {
		if ctx.board.Distance(seed, candidate) < spacing {
			return false
		}
	}
	return true
}

// Twice the steps in from the map edge, so edge spaces stay valid but score below interior ones
const blobEdgeSoftening = 2.0

// The spacing-valid candidate farthest from every seed and, more softly, the map edge (random among ties), or nil when none is valid
func farthestBlobSeed(ctx *mapScriptContext, candidates []Space, seeds []Space, spacing int, insets map[uint]int) Space {
	var best []Space
	farthest := -1.0
	for _, candidate := range candidates {
		if !blobSeedFits(ctx, candidate, seeds, spacing) {
			continue
		}
		nearest := blobEdgeSoftening * float64(insets[candidate.Key()]+1)
		for _, seed := range seeds {
			nearest = min(nearest, float64(ctx.board.Distance(seed, candidate)))
		}
		if nearest > farthest {
			best, farthest = best[:0], nearest
		}
		if nearest == farthest {
			best = append(best, candidate)
		}
	}
	if len(best) == 0 {
		return nil
	}
	return best[ctx.rng.Intn(len(best))]
}

func resolveBlobSpacing(ctx *mapScriptContext, hard ScriptExpr, soft ScriptExpr) (int, int, error) {
	min_spacing, err := hard.resolveInt(ctx.vars)
	if err != nil {
		return 0, 0, err
	}
	preferred_spacing, err := soft.resolveInt(ctx.vars)
	return min_spacing, preferred_spacing, err
}

// Each seed tries preferred_spacing first and steps down a level after repeated failures, never closer than min_spacing
func pickBlobSeeds(ctx *mapScriptContext, candidates []Space, seed_count int, min_spacing int, preferred_spacing int, evenness float64) []Space {
	seeds := make([]Space, 0, seed_count)
	insets := ctx.edgeInsets()
	target := max(min_spacing, preferred_spacing)
	spacing, failures := target, 0
	max_attempts := seed_count * (20 + (target-min_spacing)*blobSeedAttemptsPerSpacing)
	for attempt := 0; len(candidates) > 0 && len(seeds) < seed_count && attempt < max_attempts; attempt++ {
		var candidate Space
		if len(seeds) > 0 && ctx.rng.Float64() < evenness {
			candidate = farthestBlobSeed(ctx, candidates, seeds, spacing, insets)
		} else {
			candidate = candidates[ctx.rng.Intn(len(candidates))]
		}
		if candidate != nil && blobSeedFits(ctx, candidate, seeds, spacing) {
			seeds, spacing, failures = append(seeds, candidate), target, 0
		} else if failures++; failures >= blobSeedAttemptsPerSpacing && spacing > min_spacing {
			spacing, failures = spacing-1, 0
		}
	}
	return seeds
}

func resolvePoints(ctx *mapScriptContext, raw []axialParamJSON) ([]game_utils.Coordinate2D, error) {
	pts := make([]game_utils.Coordinate2D, len(raw))
	for i, p := range raw {
		x, err := p.X.resolveInt(ctx.vars)
		if err != nil {
			return nil, err
		}
		y, err := p.Y.resolveInt(ctx.vars)
		if err != nil {
			return nil, err
		}
		pts[i] = game_utils.Coordinate2D{X: x, Y: y}
	}
	return pts, nil
}

func blobSeeds(ctx *mapScriptContext, p blobPlacementJSON, candidates []Space, seed_count int, min_spacing int, preferred_spacing int) ([]Space, error) {
	if p.Pattern == "points" {
		pts, err := resolvePoints(ctx, p.Points)
		if err != nil {
			return nil, err
		}
		seeds := make([]Space, 0, len(pts))
		for _, pt := range pts {
			space := ctx.board.Space(pt)
			if space == nil {
				return nil, fmt.Errorf("blob point (%d, %d) is not on the board", pt.X, pt.Y)
			}
			seeds = append(seeds, space)
		}
		return seeds, nil
	}
	evenness, err := p.Evenness.resolve(ctx.vars)
	if err != nil {
		return nil, err
	}
	if evenness < 0 || evenness > 1 {
		return nil, fmt.Errorf("blob placement evenness must be in [0, 1], got %v", evenness)
	}
	return pickBlobSeeds(ctx, candidates, seed_count, min_spacing, preferred_spacing, evenness), nil
}

func lineSeedSpaces(ctx *mapScriptContext, pts []game_utils.Coordinate2D) ([][]Space, error) {
	if len(pts) < 2 {
		return nil, fmt.Errorf("seed_region line requires at least 2 points")
	}
	var groups [][]Space
	for i := 0; i+1 < len(pts); i += 2 {
		var group []Space
		for _, c := range hexLine(pts[i], pts[i+1]) {
			if s := ctx.board.Space(c); s != nil {
				group = append(group, s)
			}
		}
		if len(group) > 0 {
			groups = append(groups, group)
		}
	}
	return groups, nil
}

func rectSeedSpaces(ctx *mapScriptContext, pts []game_utils.Coordinate2D) ([][]Space, error) {
	if len(pts) < 2 {
		return nil, fmt.Errorf("seed_region rectangle requires at least 2 points")
	}
	var groups [][]Space
	for i := 0; i+1 < len(pts); i += 2 {
		a, b := pts[i], pts[i+1]
		minR, maxR := min(a.Y, b.Y), max(a.Y, b.Y)
		minC, maxC := min(a.X+floorDiv2(a.Y), b.X+floorDiv2(b.Y)), max(a.X+floorDiv2(a.Y), b.X+floorDiv2(b.Y))
		var group []Space
		for r := minR; r <= maxR; r++ {
			for c := minC; c <= maxC; c++ {
				if s := ctx.board.Space(game_utils.Coordinate2D{X: c - floorDiv2(r), Y: r}); s != nil {
					group = append(group, s)
				}
			}
		}
		if len(group) > 0 {
			groups = append(groups, group)
		}
	}
	return groups, nil
}

func regionSeedSpaces(ctx *mapScriptContext, region string, points []axialParamJSON) ([][]Space, error) {
	pts, err := resolvePoints(ctx, points)
	if err != nil {
		return nil, err
	}
	if region == "line" {
		return lineSeedSpaces(ctx, pts)
	}
	return rectSeedSpaces(ctx, pts)
}

func blobSeedGroups(ctx *mapScriptContext, p blobPlacementJSON, candidates []Space, seed_count int, min_spacing int, preferred_spacing int) ([][]Space, error) {
	region := p.SeedRegion
	if region == "" && (p.Pattern == "line" || p.Pattern == "rectangle" || p.Pattern == "path") {
		region = p.Pattern
	}
	if region == "path" {
		return pathSeedSpaces(ctx, p.Between, p.Points, p.PathCosts)
	}
	if region == "line" || region == "rectangle" {
		return regionSeedSpaces(ctx, region, p.Points)
	}
	seeds, err := blobSeeds(ctx, p, candidates, seed_count, min_spacing, preferred_spacing)
	if err != nil {
		return nil, err
	}
	groups := make([][]Space, len(seeds))
	for i, s := range seeds {
		groups[i] = []Space{s}
	}
	return groups, nil
}

// Reports which spaces a blob may not grow into: none, or everything outside the ring when keep_inside is set
func blobGrowthLimit(candidates []Space, keep_inside bool) func(Space) bool {
	allowed := make(map[uint]bool, len(candidates))
	for _, space := range candidates {
		allowed[space.Key()] = true
	}
	return func(space Space) bool { return keep_inside && !allowed[space.Key()] }
}

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

func pickBlobSize(rng *rand.Rand, low int, high int) int {
	if low == high {
		return low
	}
	return util.RandomIntFrom(rng, low, high)
}
