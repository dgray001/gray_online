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
	name          string
	zones         []*zone
	byLocal       [3][3]*zone
	borders       [6]*space
	// the edge of borders[d] that edge d joins; the opposite edge unless a link says otherwise
	arrivals [6]int
	sorted   []mapgen.Space
}

func newSpace(b *Board, c game_utils.Coordinate2D) *space {
	s := &space{board: b, coord: c, key: util.Pair(c.X, c.Y), terrain: defs.DefaultTerrainId}
	for d := range s.arrivals {
		s.arrivals[d] = (d + 3) % 6
	}
	for r := -1; r <= 1; r++ {
		for q := max(-1, -1-r); q <= min(1, 1-r); q++ {
			z := newZone(s, game_utils.Coordinate2D{X: q, Y: r})
			s.zones = append(s.zones, z)
			s.byLocal[q+1][r+1] = z
		}
	}
	return s
}

func (s *space) Key() uint                           { return s.key }
func (s *space) Coordinate() game_utils.Coordinate2D { return s.coord }
func (s *space) Terrain() uint32                     { return s.terrain }
func (s *space) SetTerrain(terrain_id uint32)        { s.terrain = terrain_id }
func (s *space) SetName(name string)                 { s.name = name }
func (s *space) Name() string                        { return s.name }

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
	if local.X < -1 || local.X > 1 || local.Y < -1 || local.Y > 1 {
		return nil
	}
	if z := s.byLocal[local.X+1][local.Y+1]; z != nil {
		return z
	}
	return nil
}

func (s *space) CenterZone() mapgen.Zone {
	return s.byLocal[1][1]
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
	sort.Slice(spaces, func(i, j int) bool { return spaces[i].key < spaces[j].key })
	return spaces
}

// Cached until a border changes; callers must not modify the slice
func (s *space) SortedAdjacent() []mapgen.Space {
	if s.sorted == nil {
		neighbors := s.neighbors()
		s.sorted = make([]mapgen.Space, len(neighbors))
		for i, n := range neighbors {
			s.sorted[i] = n
		}
	}
	return s.sorted
}

func (s *space) ClearOccupants() {
	for _, z := range s.zones {
		z.override, z.resource, z.building, z.units = 0, 0, nil, nil
	}
}
