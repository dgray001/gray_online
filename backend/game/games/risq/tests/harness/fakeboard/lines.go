package fakeboard

import (
	"fmt"
	"slices"

	"github.com/dgray001/gray_online/game/game_utils"
)

// Every non-border adjacency once, as "x,y>x,y:direction" with "-" for a center link, sorted
func (b *Board) LinkLines() []string {
	lines := []string{}
	for _, s := range b.order {
		for d, v := range game_utils.AxialDirectionVectors() {
			if n := s.borders[d]; n != nil && n.coord != *s.coord.Add(&v) && s.key < n.key {
				lines = append(lines, fmt.Sprintf("%d,%d>%d,%d:%d", s.coord.X, s.coord.Y, n.coord.X, n.coord.Y, d))
			}
		}
		for _, n := range s.links {
			if s.key < n.key {
				lines = append(lines, fmt.Sprintf("%d,%d>%d,%d:-", s.coord.X, s.coord.Y, n.coord.X, n.coord.Y))
			}
		}
	}
	slices.Sort(lines)
	return lines
}

// Each region as "name|bonus|sorted space keys", sorted
func (b *Board) RegionLines() []string {
	lines := []string{}
	for _, region := range b.Regions {
		keys := make([]uint, 0, len(region.Keys))
		for key := range region.Keys {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		lines = append(lines, fmt.Sprintf("%s|%v|%v", region.Name, region.GoldBonus, keys))
	}
	slices.Sort(lines)
	return lines
}
