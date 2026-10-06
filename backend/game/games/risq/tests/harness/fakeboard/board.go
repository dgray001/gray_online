// Package fakeboard is an in-memory mapgen.Board that mirrors the engine's ordering and adjacency rules.
package fakeboard

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/mapgen"
)

type Board struct {
	size       uint16
	order      []*space
	byCoord    map[game_utils.Coordinate2D]*space
	byKey      map[uint]*space
	distances  map[*space][]uint16
	Regions    []Region
	Banks      map[int]mapgen.StartingBank
	Techs      map[int][]uint32
	Unlimited  bool
	GoldIncome *float64
	NeedRegion *bool
	Violations []string
}

type Region struct {
	Name      string
	GoldBonus float64
	Keys      map[uint]bool
}

func New() *Board {
	return &Board{Banks: map[int]mapgen.StartingBank{}, Techs: map[int][]uint32{}}
}

func (b *Board) BoardSize() uint16 {
	return b.size
}

func (b *Board) Spaces() []mapgen.Space {
	spaces := make([]mapgen.Space, len(b.order))
	for i, s := range b.order {
		spaces[i] = s
	}
	return spaces
}

func (b *Board) Space(c game_utils.Coordinate2D) mapgen.Space {
	if s, ok := b.byCoord[c]; ok {
		return s
	}
	return nil
}
