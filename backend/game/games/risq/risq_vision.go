package risq

/**
Visibility Levels (all ∈ this range):
  0: unexplored: nothing visible
  1: not visible: terrain, ownership (no zones)
  2: poor visibility, total resources / buildings / units (no zones)
  3: modest visibility, resources / buildings / units in zones (no stats)
  4: good visibility, full stats of all resources / buildings / units
  5: spies, can see enemy orders

Inequalities:
  space >= edge_adjacent
  edge_adjacent >= adjacent
  adjacent >= edge_opposite
  edge_opposite >= secondary
*/

type RisqVision struct {
	space         uint8 // the space you are in
	edge_adjacent uint8 // space directly adjacent to edge zone
	adjacent      uint8 // adjacent vision if in center zone or side spaces if in edge zone
	edge_opposite uint8 // 3 spaces opposite to edge zone
	secondary     uint8 // second ring of spaces
}
