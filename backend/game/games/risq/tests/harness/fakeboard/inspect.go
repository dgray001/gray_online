package fakeboard

import (
	"fmt"
	"strings"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/mapgen"
)

var _ mapgen.Board = (*Board)(nil)

type ZoneInfo struct {
	Local    game_utils.Coordinate2D
	Resource uint32
	// zero ID means no building
	Building Building
	Units    []Unit
	Override uint32
}

type SpaceInfo struct {
	Coord    game_utils.Coordinate2D
	Terrain  uint32
	Passable bool
	Zones    []ZoneInfo
}

// Every space in allocation order with its zones in engine order
func (b *Board) Inspect() []SpaceInfo {
	infos := make([]SpaceInfo, len(b.order))
	for i, s := range b.order {
		infos[i] = SpaceInfo{Coord: s.coord, Terrain: s.terrain, Passable: !s.Impassable()}
		infos[i].Zones = make([]ZoneInfo, len(s.zones))
		for j, z := range s.zones {
			info := ZoneInfo{Local: z.local, Resource: z.resource, Units: z.units, Override: z.override}
			if z.building != nil {
				info.Building = *z.building
			}
			infos[i].Zones[j] = info
		}
	}
	return infos
}

// Everything a script decided, as one comparable string
func (b *Board) Dump() string {
	var dump strings.Builder
	fmt.Fprintf(&dump, "size=%d banks=%v techs=%v unlimited=%t\n", b.size, b.Banks, b.Techs, b.Unlimited)
	for _, info := range b.Inspect() {
		fmt.Fprintf(&dump, "%v t%d %v\n", info.Coord, info.Terrain, info.Zones)
	}
	for _, region := range b.Regions {
		fmt.Fprintf(&dump, "region %s %v %v\n", region.Name, region.GoldBonus, region.Keys)
	}
	return dump.String()
}

func (b *Board) SetBackgroundImage(name string, tl [2]int, tr [2]int) {}
