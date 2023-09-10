package fiddlesticks

import "github.com/dgray001/gray_online/game"

/*
   ========================
   >>>>> FIDDLESTICKS <<<<<
	 ========================

	 Objective: end the game with the most points
	 Description: earn the exact number of tricks you bet to earn points for that round.
	  The number of cards per player will increase by one each round until there's not enough cards to deal.
		After that decrease the number of cards by one to finish the second half of the game.
	 Betting: starting to left of dealer, players bet on the number of tricks they think they will win.
	 Tricks: play starts to left of dealer, trump is next card in deck, must follow suit, winner starts next trick, etc.
	 Scoring: a player only scores points if they win the exact number of tricks they bet
*/

type GameFiddlesticks struct {
	game              *game.GameBase
	players           []*FiddlesticksPlayer
	deck              *game.StandardDeck
	round             uint8
	max_round         uint8
	rounds_increasing bool
	dealer            int
	turn              int
	betting           bool
	// channel for actions
}

type FiddlesticksPlayer struct {
	player *game.Player
	cards  []*game.StandardCard
	score  uint16
	bet    uint8
	tricks uint8
}

type ActionType uint8

const (
	BET ActionType = iota
	PLAY_CARD
)

type FiddlesticksAction struct {
	player      *FiddlesticksPlayer
	action_type ActionType
	action_data uint8
}

func (f *GameFiddlesticks) CreateGame(g *game.GameBase) *GameFiddlesticks {
	fiddlesticks := GameFiddlesticks{
		game:              g,
		players:           []*FiddlesticksPlayer{},
		deck:              game.CreateStandardDeck(),
		round:             0,
		rounds_increasing: true,
		dealer:            -1,
		turn:              -1,
		betting:           false,
	}
	for _, player := range g.Players {
		fiddlesticks.players = append(fiddlesticks.players, &FiddlesticksPlayer{
			player: player,
			cards:  []*game.StandardCard{},
			score:  0,
		})
	}
	if len(fiddlesticks.players) < 2 {
		panic("Need at least two players to play fiddlesticks")
	}
	fiddlesticks.max_round = uint8((fiddlesticks.deck.Size() - 1) / len(fiddlesticks.players))
	return &fiddlesticks
}

func (f *GameFiddlesticks) StartGame() {
	f.game.StartGame()
	f.dealNextRound()
}

func (f *GameFiddlesticks) dealNextRound() {
	if f.round == 1 && !f.rounds_increasing {
		// TODO: Game ends
		return
	}
	if f.rounds_increasing {
		if f.round == f.max_round {
			f.rounds_increasing = false
			f.round--
		} else {
			f.round++
		}
	} else {
		f.round--
	}
	f.deck.Reset()
	dealt_cards := f.deck.DealCards(uint8(len(f.players)), f.round)
	for i := 0; i < len(f.players); i++ {
		cards := dealt_cards[i]
		i += f.dealer
		if i > len(f.players) {
			i -= len(f.players)
		}
		f.players[i].cards = cards
		f.players[i].tricks = 0
	}
	f.betting = true
	f.turn = f.dealer + 1
	if f.turn > len(f.players) {
		f.turn -= len(f.players)
	}
}
