package mapgen

import (
	"math/rand"

	"github.com/dgray001/gray_online/game/game_utils"
)

// The live board a map script shapes; every mutation applies immediately, so rng draws and internal id
// allocation happen in exactly script order
type Board interface {
	Allocate(board_size uint16)
	BoardSize() uint16
	Spaces() []Space
	// nil when the coordinate is off the board or was carved out by the shape
	Space(c game_utils.Coordinate2D) Space
	RemoveSpace(s Space)
	AddRegion(name string, coordinate_keys map[uint]bool) error
	PlaceResource(z Zone, resource_id uint32)
	// false when the zone refused the building (occupied)
	PlaceBuilding(z Zone, building_id uint32, player_index int) bool
	PlaceUnit(z Zone, unit_id uint32, player_index int)
}

type Space interface {
	Key() uint
	Coordinate() game_utils.Coordinate2D
	Terrain() uint32
	SetTerrain(terrain_id uint32)
	Impassable() bool
	SortedAdjacent() []Space
	Zones() []Zone
	// nil when local isn't one of the space's seven zones
	Zone(local game_utils.Coordinate2D) Zone
	CenterZone() Zone
	ShuffledEdgeZones(rng *rand.Rand) []Zone
	// removes resources and buildings and resets zone terrain overrides
	ClearOccupants()
}

type Zone interface {
	Key() uint
	Local() game_utils.Coordinate2D
	Space() Space
	IsCenter() bool
	Adjacent() []Zone
	Occupied() bool
	ResourceId() (uint32, bool)
	RemoveResource()
	SetTerrainOverride(terrain_id uint32)
}
