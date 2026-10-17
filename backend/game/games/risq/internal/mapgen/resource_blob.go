package mapgen

import (
	"encoding/json"
	"fmt"
	"math/rand"
)

type resourceBlobParams struct {
	ResourceId       uint32             `json:"resource_id"`
	CategoryWeights  map[string]float64 `json:"category_weights,omitempty"`
	ResourceWeights  map[string]float64 `json:"resource_weights,omitempty"`
	SeedCount        ScriptExpr         `json:"seed_count"`
	Size             BlobSize           `json:"size"`
	MinSpacing       ScriptExpr         `json:"min_spacing"`
	Roundness        ScriptExpr         `json:"roundness"`
	PreferredSpacing ScriptExpr         `json:"preferred_spacing"`
	Placement        blobPlacementJSON  `json:"placement"`
	Zone             string             `json:"zone,omitempty"`
	On               terrainFilterJSON  `json:"on"`
	// replaces existing resources in the blob's zones instead of growing around them
	Overwrite bool `json:"overwrite"`
}

// A random zone in space that matches the zone selector and is free, or overwritable, or nil when there is none
func freeResourceZone(space Space, selector string, overwrite bool, rng *rand.Rand) Zone {
	free := make([]Zone, 0)
	for _, z := range space.Zones() {
		if (overwrite || !z.Occupied()) && resourceZoneMatches(z, selector) {
			free = append(free, z)
		}
	}
	if len(free) == 0 {
		return nil
	}
	return free[rng.Intn(len(free))]
}

// Grows one blob of resources per seed over the zone graph; each blob keeps one category or id pool
func stepResourceBlob(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[resourceBlobParams](raw, "resource_blob")
	if err != nil {
		return err
	}
	if !validateResourceZone(p.Zone) {
		return fmt.Errorf("resource_blob: invalid zone %q", p.Zone)
	}
	entries, err := placeEntries(ctx, p.ResourceId, p.CategoryWeights, p.ResourceWeights)
	if err != nil {
		return err
	}
	seed_count, err := p.SeedCount.resolveInt(ctx.vars)
	if err != nil {
		return err
	}
	min_size, max_size, err := p.Size.resolve(ctx.vars)
	if err != nil {
		return err
	}
	min_spacing, preferred_spacing, err := resolveBlobSpacing(ctx, p.MinSpacing, p.PreferredSpacing)
	if err != nil {
		return err
	}
	roundness := defaultBlobRoundness
	if p.Roundness.provided() {
		if roundness, err = p.Roundness.resolve(ctx.vars); err != nil {
			return err
		}
		if roundness < 0 || roundness > 1 {
			return fmt.Errorf("resource_blob: roundness must be in [0, 1], got %v", roundness)
		}
	}
	candidates, err := blobSeedCandidates(ctx, p.Placement)
	if err != nil {
		return err
	}
	matches, err := p.On.matcher(ctx)
	if err != nil {
		return err
	}
	seeds, err := blobSeeds(ctx, p.Placement, filterSpaces(candidates, matches), seed_count, min_spacing, preferred_spacing)
	if err != nil {
		return err
	}
	excluded := func(z Zone) bool {
		return (!p.Overwrite && z.Occupied()) || z.Space().Impassable() || !matches(z.Space()) || !resourceZoneMatches(z, p.Zone)
	}
	for _, seed := range seeds {
		start := freeResourceZone(seed, p.Zone, p.Overwrite, ctx.rng)
		if start == nil {
			continue
		}
		ids := pickScatterIds(entries, ctx.rng)
		for _, z := range growBlob([]Zone{start}, pickBlobSize(ctx.rng, min_size, max_size), roundness, ctx.rng, zoneKey, zoneNeighbors, excluded) {
			if z.Occupied() {
				z.RemoveResource()
			}
			ctx.board.PlaceResource(z, ids[ctx.rng.Intn(len(ids))])
		}
	}
	return nil
}

func zoneKey(z Zone) uint { return z.Key() }

func zoneNeighbors(z Zone) []Zone { return z.Adjacent() }
