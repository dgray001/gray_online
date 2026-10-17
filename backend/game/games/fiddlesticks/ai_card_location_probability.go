package fiddlesticks

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dgray001/gray_online/game/game_utils"
)

const KITTY_COLUMN = -1

/** describes the probability that a card is in any possible location */
type CardLocationProbability struct {
	card      *game_utils.StandardCard
	played    bool
	hand      bool
	trump     bool
	unknown   bool
	kitty     float64
	player    []float64 // indexed by player id
	opponents []int     // ids of the players the model can't see
}

func (p *CardLocationProbability) reset(players []*FiddlesticksPlayer, model_player int) {
	p.played = false
	p.hand = false
	p.trump = false
	p.unknown = false
	p.kitty = 0
	p.player = make([]float64, len(players))
	p.opponents = p.opponents[:0]
	for i, player := range players {
		if i == model_player {
			continue
		}
		p.opponents = append(p.opponents, player.player.Player_id)
	}
}

func (p *CardLocationProbability) inHand() {
	p.hand = true
}

func (p *CardLocationProbability) isTrump() {
	p.trump = true
}

func (p *CardLocationProbability) isUnknown(prob_kitty float64, prob_hand float64) {
	p.unknown = true
	p.kitty = prob_kitty
	for _, id := range p.opponents {
		p.player[id] = prob_hand
	}
}

func (p *CardLocationProbability) cardPlayed() {
	p.played = true
	p.hand = false
	p.unknown = false
	p.kitty = 0
	for _, id := range p.opponents {
		p.player[id] = 0
	}
}

func (p *CardLocationProbability) column(id int) float64 {
	if id == KITTY_COLUMN {
		return p.kitty
	}
	return p.player[id]
}

func (p *CardLocationProbability) copyLocations(other *CardLocationProbability) {
	p.kitty = other.kitty
	copy(p.player, other.player)
}

func (p *CardLocationProbability) setColumn(id int, value float64) {
	if id == KITTY_COLUMN {
		p.kitty = value
		return
	}
	p.player[id] = value
}

func (p *CardLocationProbability) scaleColumn(id int, factor float64) {
	if id == KITTY_COLUMN {
		p.kitty *= factor
		return
	}
	p.player[id] *= factor
}

func (p *CardLocationProbability) normalize() {
	total := p.kitty
	for _, id := range p.opponents {
		total += p.player[id]
	}
	if total == 0 {
		return
	}
	p.kitty /= total
	for _, id := range p.opponents {
		p.player[id] /= total
	}
}

func (p *CardLocationProbability) printString() string {
	players := make([]string, 0, len(p.opponents))
	for _, id := range p.opponents {
		players = append(players, strconv.FormatFloat(p.player[id], 'f', 3, 64))
	}
	return fmt.Sprintf("%s: %t, %t, %t, %.3f, (%s)", p.card.GetShortName(), p.played, p.hand, p.trump, p.kitty, strings.Join(players, ", "))
}
