package defs

/**
Visibility Levels (all ∈ this range):
  0: unexplored: nothing visible
  1: fog of war: zones visible, buildings/resources are a cached last-known snapshot, no units
  2: poor visibility: live buildings/resources with full stats, units visible as a count only (no type/stats)
  3: good visibility: full unit detail (type, stats)
  4: spies: can see enemy orders

Inequalities:
  space >= edge_adjacent
  edge_adjacent >= adjacent
  adjacent >= edge_opposite
  edge_opposite >= secondary
*/

type VisibilityLevel = uint8

const (
	VisibilityUnexplored VisibilityLevel = iota
	VisibilityFog
	VisibilityPoor
	VisibilityGood
	VisibilitySpy
)

type RisqVision struct {
	Space         uint8 // the space you are in
	Edge_adjacent uint8 // space directly adjacent to edge zone
	Adjacent      uint8 // adjacent vision if in center zone or side spaces if in edge zone
	Edge_opposite uint8 // 3 spaces opposite to edge zone
	Secondary     uint8 // second ring of spaces
}

var defaultRisqVision = RisqVision{
	Space:         VisibilityGood,
	Edge_adjacent: VisibilityGood,
	Adjacent:      VisibilityPoor,
	Edge_opposite: VisibilityUnexplored,
	Secondary:     VisibilityUnexplored,
}

var defaultHillyVision = RisqVision{
	Space:         VisibilityGood,
	Edge_adjacent: VisibilityGood,
	Adjacent:      VisibilityGood,
	Edge_opposite: VisibilityPoor,
	Secondary:     VisibilityUnexplored,
}

var defaultMountainousVision = RisqVision{
	Space:         VisibilityGood,
	Edge_adjacent: VisibilityGood,
	Adjacent:      VisibilityGood,
	Edge_opposite: VisibilityGood,
	Secondary:     VisibilityPoor,
}

var allGoodVision = RisqVision{
	Space:         VisibilityGood,
	Edge_adjacent: VisibilityGood,
	Adjacent:      VisibilityGood,
	Edge_opposite: VisibilityGood,
	Secondary:     VisibilityGood,
}

func boostVisionTier(base uint8, target uint8) uint8 {
	if target <= base {
		return base
	}
	if base == VisibilityUnexplored {
		return min(target, VisibilityPoor)
	}
	return min(target, base+1)
}

func (v RisqVision) Boost(target RisqVision) RisqVision {
	return RisqVision{
		Space:         boostVisionTier(v.Space, target.Space),
		Edge_adjacent: boostVisionTier(v.Edge_adjacent, target.Edge_adjacent),
		Adjacent:      boostVisionTier(v.Adjacent, target.Adjacent),
		Edge_opposite: boostVisionTier(v.Edge_opposite, target.Edge_opposite),
		Secondary:     boostVisionTier(v.Secondary, target.Secondary),
	}
}

func (v RisqVision) WithTerrain(terrain TerrainType) RisqVision {
	has_custom := v != defaultRisqVision
	switch terrain {
	case TerrainType_HILLY:
		if !has_custom {
			return defaultHillyVision
		}
		return v.Boost(defaultHillyVision)
	case TerrainType_MOUNTAINOUS:
		if !has_custom {
			return defaultMountainousVision
		}
		return v.Boost(allGoodVision)
	default:
		return v
	}
}

type risqVisionJSON struct {
	Space        uint8 `json:"space"`
	EdgeAdjacent uint8 `json:"edge_adjacent"`
	Adjacent     uint8 `json:"adjacent"`
	EdgeOpposite uint8 `json:"edge_opposite"`
	Secondary    uint8 `json:"secondary"`
}

func (j risqVisionJSON) toRisqVision() RisqVision {
	return RisqVision{
		Space:         j.Space,
		Edge_adjacent: j.EdgeAdjacent,
		Adjacent:      j.Adjacent,
		Edge_opposite: j.EdgeOpposite,
		Secondary:     j.Secondary,
	}
}

func resolveVision(j *risqVisionJSON) RisqVision {
	if j == nil {
		return defaultRisqVision
	}
	return j.toRisqVision()
}
