package fiddlesticks

import (
	"strconv"

	"github.com/dgray001/gray_online/game"
	"github.com/gin-gonic/gin"
)

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

func (p *FiddlesticksPlayer) toFrontend() gin.H {
	player := gin.H{
		"score":  strconv.Itoa(int(p.score)),
		"bet":    strconv.Itoa(int(p.bet)),
		"tricks": strconv.Itoa(int(p.tricks)),
	}
	if p.player != nil {
		player["player"] = p.player.ToFrontend()
	}
	cards := []gin.H{}
	for _, card := range p.cards {
		if card != nil {
			cards = append(cards, card.ToFrontend())
		}
	}
	player["cards"] = cards
	return player
}

type ActionType uint8

const (
	UNDEFINED_ACTIONTYPE ActionType = iota
	BET
	PLAY_CARD
)

type FiddlesticksAction struct {
	player      *FiddlesticksPlayer
	action_type ActionType
	action_data uint8
}

func CreateGame(g *game.GameBase) *GameFiddlesticks {
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
		panic("Need at least two players to play fiddlesticks") // TODO: remove panics
	}
	fiddlesticks.max_round = uint8((fiddlesticks.deck.Size() - 1) / len(fiddlesticks.players))
	return &fiddlesticks
}

func (f *GameFiddlesticks) GetId() uint64 {
	return f.game.Game_id
}

func (f *GameFiddlesticks) StartGame() {
	f.game.StartGame()
	f.dealNextRound()
}

func (f *GameFiddlesticks) Valid() bool {
	if f.game == nil || f.deck == nil {
		return false
	}
	return true
}

func (f *GameFiddlesticks) ToFrontend() gin.H {
	game := gin.H{
		"round":             strconv.Itoa(int(f.round)),
		"max_round":         strconv.Itoa(int(f.max_round)),
		"rounds_increasing": strconv.FormatBool(f.rounds_increasing),
		"dealer":            strconv.Itoa(f.dealer),
		"turn":              strconv.Itoa(f.turn),
		"betting":           strconv.FormatBool(f.betting),
	}
	if f.game != nil {
		game["game_base"] = f.game.ToFrontend()
	}
	if f.deck != nil {
		game["deck"] = f.deck.ToFrontend()
	}
	players := []gin.H{}
	for _, player := range f.players {
		if player != nil {
			players = append(players, player.toFrontend())
		}
	}
	game["players"] = players
	return game
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
