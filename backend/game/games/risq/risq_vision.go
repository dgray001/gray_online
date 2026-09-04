package risq

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
	space         uint8 // the space you are in
	edge_adjacent uint8 // space directly adjacent to edge zone
	adjacent      uint8 // adjacent vision if in center zone or side spaces if in edge zone
	edge_opposite uint8 // 3 spaces opposite to edge zone
	secondary     uint8 // second ring of spaces
}

var defaultRisqVision = RisqVision{
	space:         VisibilityGood,
	edge_adjacent: VisibilityGood,
	adjacent:      VisibilityPoor,
	edge_opposite: VisibilityFog,
	secondary:     VisibilityFog,
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
		space:         j.Space,
		edge_adjacent: j.EdgeAdjacent,
		adjacent:      j.Adjacent,
		edge_opposite: j.EdgeOpposite,
		secondary:     j.Secondary,
	}
}

func resolveVision(j *risqVisionJSON) RisqVision {
	if j == nil {
		return defaultRisqVision
	}
	return j.toRisqVision()
}

func showOrdersTo(subject_player_id int, zone *RisqZone, viewer_player_id int) bool {
	if subject_player_id == viewer_player_id {
		return true
	}
	if zone == nil || zone.space == nil {
		return false
	}
	return zone.space.getVisibility(viewer_player_id) >= VisibilitySpy
}
