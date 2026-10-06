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
	AddRegion(name string, gold_bonus float64, coordinate_keys map[uint]bool) error
	// 0 to 5 joins a's edge in that direction to b's opposite edge like a normal border
	ConnectSpaces(a, b Space, direction int) error
	// space-to-space steps over the live space graph, so seams and links count as one step
	Distance(a, b Space) int
	PlaceResource(z Zone, resource_id uint32)
	// false when the zone refused the building (occupied)
	PlaceBuilding(z Zone, building_id uint32, player_index int) bool
	PlaceUnit(z Zone, unit_id uint32, player_index int)
	// replaces a player's starting stockpile
	SetStartingBank(player_index int, bank StartingBank)
	// researches a tech for a player before turn 1, unlocking what finishing it would
	GrantStartingTech(player_index int, tech_id uint32) error
	// removes every player's population cap
	SetUnlimitedPopulation()
	// gold paid per turn to the owner of each space
	SetSpaceGoldIncome(gold float64)
	// whether hiring a mercenary into a space needs the player to hold that space's whole region; true by default
	SetMercenariesNeedRegion(need bool)
	SetBackgroundImage(name string, tl [2]int, tr [2]int)
}

type StartingBank struct {
	Food  float64 `json:"food"`
	Wood  float64 `json:"wood"`
	Stone float64 `json:"stone"`
	Gold  float64 `json:"gold"`
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
	// removes units, resources and buildings and resets zone terrain overrides
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
