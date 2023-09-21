package fiddlesticks

import (
	"fmt"

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
	trump             *game.StandardCard
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
		"score":  p.score,
		"bet":    p.bet,
		"tricks": p.tricks,
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
	var player_id = 0
	for _, player := range g.Players {
		player.Player_id = player_id
		fiddlesticks.players = append(fiddlesticks.players, &FiddlesticksPlayer{
			player: player,
			cards:  []*game.StandardCard{},
			score:  0,
		})
		player_id++
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

func (f *GameFiddlesticks) GetBase() *game.GameBase {
	return f.game
}

func (f *GameFiddlesticks) StartGame() {
	f.game.StartGame()
	f.dealNextRound()
}

func (f *GameFiddlesticks) PlayerAction(action game.PlayerAction) {
	fmt.Println("player action", action.Kind, action.ClientId, action.Action)
	player := f.game.Players[uint64(action.ClientId)]
	if player == nil {
		fmt.Println("Invalid client id", action.ClientId)
		return
	}
	player_id := player.Player_id
	switch action.Kind {
	case "bet":
		if player_id != f.turn {
			fmt.Println("Not", player_id, "player's turn but", f.turn, "player's turn")
			return
		}
		if !f.betting {
			fmt.Println("Not currently betting")
			return
		}
		bet_value, ok := action.Action["amount"].(uint8)
		if !ok {
			fmt.Println("Bet value invalid")
			return
		}
		f.players[f.turn].bet = bet_value
		f.turn++
		if f.turn >= len(f.players) {
			f.turn -= len(f.players)
		}
		f.broadcastUpdate(&game.UpdateMessage{Kind: "bet", Content: gin.H{
			"amount":    bet_value,
			"player_id": player_id,
		}})
	default:
		fmt.Println("Unknown game update type", action.Kind)
	}
}

func (f *GameFiddlesticks) Valid() bool {
	if f.game == nil || f.deck == nil {
		return false
	}
	return true
}

func (f *GameFiddlesticks) ToFrontend() gin.H {
	game := gin.H{
		"round":             f.round,
		"max_round":         f.max_round,
		"rounds_increasing": f.rounds_increasing,
		"dealer":            f.dealer,
		"turn":              f.turn,
		"betting":           f.betting,
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
	if f.trump != nil {
		game["trump"] = f.trump.ToFrontend()
	}
	return game
}

func (f *GameFiddlesticks) broadcastUpdate(update *game.UpdateMessage) {
	for _, player := range f.players {
		player.player.AddUpdate(update)
	}
}

func (f *GameFiddlesticks) dealNextRound() {
	if f.round == 1 && !f.rounds_increasing {
		// TODO: Game ends
		return
	}
	f.dealer++
	if f.dealer >= len(f.players) {
		f.dealer = 0
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
		j := i + f.dealer
		if j >= len(f.players) {
			j -= len(f.players)
		}
		f.players[j].cards = cards
		f.players[j].tricks = 0
	}
	f.trump = f.deck.DrawCard()
	for _, player := range f.players {
		frontend_cards := []gin.H{}
		for _, card := range player.cards {
			frontend_cards = append(frontend_cards, card.ToFrontend())
		}
		player.player.AddUpdate(&game.UpdateMessage{Kind: "deal-round", Content: gin.H{
			"dealer": f.dealer,
			"round":  f.round,
			"trump":  f.trump.ToFrontend(),
			"cards":  frontend_cards,
		}})
	}
	f.betting = true
	f.turn = f.dealer + 1
	if f.turn >= len(f.players) {
		f.turn -= len(f.players)
	}
}
