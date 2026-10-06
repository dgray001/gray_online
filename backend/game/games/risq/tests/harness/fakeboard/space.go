package fakeboard

import (
	"math/rand"
	"sort"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/internal/mapgen"
	"github.com/dgray001/gray_online/util"
)

type space struct {
	board         *Board
	coord         game_utils.Coordinate2D
	key           uint
	distanceIndex int
	terrain       uint32
	zones         []*zone
	byLocal       map[game_utils.Coordinate2D]*zone
	borders       [6]*space
	links         []*space
}

func newSpace(b *Board, c game_utils.Coordinate2D) *space {
	s := &space{board: b, coord: c, key: util.Pair(c.X, c.Y), terrain: defs.DefaultTerrainId, byLocal: map[game_utils.Coordinate2D]*zone{}}
	for r := -1; r <= 1; r++ {
		for q := max(-1, -1-r); q <= min(1, 1-r); q++ {
			z := newZone(s, game_utils.Coordinate2D{X: q, Y: r})
			s.zones = append(s.zones, z)
			s.byLocal[z.local] = z
		}
	}
	return s
}

func (s *space) Key() uint                           { return s.key }
func (s *space) Coordinate() game_utils.Coordinate2D { return s.coord }
func (s *space) Terrain() uint32                     { return s.terrain }
func (s *space) SetTerrain(terrain_id uint32)        { s.terrain = terrain_id }

func (s *space) Impassable() bool {
	return defs.TerrainConfigs[s.terrain].Terrain_type.MoveCost().Impassable
}

func (s *space) Zones() []mapgen.Zone {
	zones := make([]mapgen.Zone, len(s.zones))
	for i, z := range s.zones {
		zones[i] = z
	}
	return zones
}

func (s *space) Zone(local game_utils.Coordinate2D) mapgen.Zone {
	if z, ok := s.byLocal[local]; ok {
		return z
	}
	return nil
}

func (s *space) CenterZone() mapgen.Zone {
	return s.byLocal[game_utils.Coordinate2D{}]
}

func (s *space) ShuffledEdgeZones(rng *rand.Rand) []mapgen.Zone {
	edges := make([]*zone, 0, 6)
	for _, z := range s.zones {
		if !z.IsCenter() {
			edges = append(edges, z)
		}
	}
	util.ShuffleFrom(rng, edges)
	zones := make([]mapgen.Zone, len(edges))
	for i, z := range edges {
		zones[i] = z
	}
	return zones
}

func (s *space) neighbors() []*space {
	var spaces []*space
	for _, border := range s.borders {
		if border != nil {
			spaces = append(spaces, border)
		}
	}
	spaces = append(spaces, s.links...)
	sort.Slice(spaces, func(i, j int) bool { return spaces[i].key < spaces[j].key })
	return spaces
}

func (s *space) SortedAdjacent() []mapgen.Space {
	neighbors := s.neighbors()
	spaces := make([]mapgen.Space, len(neighbors))
	for i, n := range neighbors {
		spaces[i] = n
	}
	return spaces
}

func (s *space) ClearOccupants() {
	for _, z := range s.zones {
		z.override, z.resource, z.building, z.units = 0, 0, nil, nil
	}
}
