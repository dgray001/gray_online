package fakeboard

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/mapgen"
	"github.com/dgray001/gray_online/util"
)

type Building struct {
	ID     uint32
	Player int
}

type Unit struct {
	ID     uint32
	Player int
}

type zone struct {
	space    *space
	local    game_utils.Coordinate2D
	key      uint
	resource uint32
	building *Building
	units    []Unit
	override uint32
}

func newZone(s *space, local game_utils.Coordinate2D) *zone {
	return &zone{space: s, local: local, key: util.Pair(int(s.key), int(util.Pair(local.X, local.Y)))}
}

func (z *zone) Key() uint                            { return z.key }
func (z *zone) Local() game_utils.Coordinate2D       { return z.local }
func (z *zone) Space() mapgen.Space                  { return z.space }
func (z *zone) IsCenter() bool                       { return z.local == game_utils.Coordinate2D{} }
func (z *zone) Occupied() bool                       { return z.resource != 0 || z.building != nil }
func (z *zone) SetTerrainOverride(terrain_id uint32) { z.override = terrain_id }

func (z *zone) ResourceId() (uint32, bool) {
	return z.resource, z.resource != 0
}

func (z *zone) RemoveResource() {
	z.resource = 0
}

// The center touches the six edges and any link; an edge touches the center, its edge neighbors and the facing edge across a border
func (z *zone) Adjacent() []mapgen.Zone {
	var zones []*zone
	if !z.IsCenter() {
		zones = append(zones, z.space.byLocal[game_utils.Coordinate2D{}])
	}
	for _, other := range z.space.zones {
		if other != z && !other.IsCenter() && (z.IsCenter() || game_utils.AxialDistance(z.local, other.local) == 1) {
			zones = append(zones, other)
		}
	}
	if z.IsCenter() {
		for _, linked := range z.space.links {
			zones = append(zones, linked.byLocal[game_utils.Coordinate2D{}])
		}
	} else if across := z.across(); across != nil {
		zones = append(zones, across)
	}
	adjacent := make([]mapgen.Zone, len(zones))
	for i, other := range zones {
		adjacent[i] = other
	}
	return adjacent
}

// The facing edge zone in the neighboring space, when this edge has a border
func (z *zone) across() *zone {
	direction := game_utils.AxialDirectionIndex(z.local)
	if neighbor := z.space.borders[direction]; neighbor != nil {
		return neighbor.byLocal[*z.local.Invert()]
	}
	return nil
}
