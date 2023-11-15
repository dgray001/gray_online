package fiddlesticks

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
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
	deck              *game_utils.StandardDeck
	round             uint8
	max_round         uint8
	rounds_increasing bool
	dealer            int
	turn              int
	betting           bool
	trump             *game_utils.StandardCard
	trick_leader      int
	trick             []*game_utils.StandardCard
	round_points      uint16
	trick_points      uint16
}

type FiddlesticksPlayer struct {
	player       *game.Player
	cards        []*game_utils.StandardCard
	cards_played []int
	score        uint16
	bet          uint8
	tricks       uint8
}

func (p *FiddlesticksPlayer) toFrontend(show_updates bool) gin.H {
	player := gin.H{
		"score":        p.score,
		"bet":          p.bet,
		"tricks":       p.tricks,
		"cards_played": p.cards_played,
	}
	if p.player != nil {
		player["player"] = p.player.ToFrontend(show_updates)
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

func CreateGame(g *game.GameBase) (*GameFiddlesticks, error) {
	fiddlesticks := GameFiddlesticks{
		game:              g,
		players:           []*FiddlesticksPlayer{},
		deck:              game_utils.CreateStandardDeck(),
		round:             0,
		rounds_increasing: true,
		dealer:            -1,
		turn:              -1,
		betting:           false,
		trump:             nil,
		trick_leader:      -1,
		trick:             []*game_utils.StandardCard{},
		round_points:      10,
		trick_points:      1,
	}
	var player_id = 0
	for _, player := range g.Players {
		player.Player_id = player_id
		fiddlesticks.players = append(fiddlesticks.players, &FiddlesticksPlayer{
			player:       player,
			cards:        []*game_utils.StandardCard{},
			cards_played: []int{},
			score:        0,
		})
		player_id++
	}
	if len(fiddlesticks.players) < 2 {
		return nil, errors.New("Need at least two players to play fiddlesticks")
	}
	fiddlesticks.max_round = uint8((fiddlesticks.deck.Size() - 1) / len(fiddlesticks.players))
	max_round_float, max_round_ok := g.GameSpecificSettings["max_round"].(float64)
	if max_round_ok {
		max_round := uint8(max_round_float)
		if max_round > 0 && max_round < fiddlesticks.max_round {
			fiddlesticks.max_round = max_round
		}
	}
	round_points_float, round_points_ok := g.GameSpecificSettings["round_points"].(float64)
	if round_points_ok {
		round_points := uint16(round_points_float)
		if round_points >= 0 && round_points < 100 {
			fiddlesticks.round_points = round_points
		}
	}
	trick_points_float, trick_points_ok := g.GameSpecificSettings["trick_points"].(float64)
	if trick_points_ok {
		trick_points := uint16(trick_points_float)
		if trick_points >= 0 && trick_points < 100 {
			fiddlesticks.trick_points = trick_points
		}
	}
	if fiddlesticks.round_points == 0 && fiddlesticks.trick_points == 0 {
		fiddlesticks.round_points = 1
	}
	return &fiddlesticks, nil
}

func (f *GameFiddlesticks) GetBase() *game.GameBase {
	return f.game
}

func (f *GameFiddlesticks) StartGame() {
	f.dealNextRound()
}

func (f *GameFiddlesticks) Valid() bool {
	if f.game == nil || f.deck == nil {
		return false
	}
	return true
}

func (f *GameFiddlesticks) PlayerAction(action game.PlayerAction) {
	fmt.Println("player action:", action.Kind, action.Client_id, action.Action)
	player := f.game.Players[uint64(action.Client_id)]
	if player == nil {
		fmt.Println("Invalid client id", action.Client_id)
		return
	}
	player_id := player.Player_id
	switch action.Kind {
	case "bet":
		if player_id != f.turn {
			message := fmt.Sprintf("Not %d player's turn but %d player's turn", player_id, f.turn)
			fmt.Println(message)
			f.players[player_id].player.AddFailedUpdate(&game.UpdateMessage{Kind: "bet-failed",
				Content: gin.H{"message": message, "player_id": player_id}})
			return
		}
		if !f.betting {
			fmt.Println("Not currently betting")
			return
		}
		bet_value_float, ok := action.Action["amount"].(float64)
		if !ok {
			message := fmt.Sprintf("Bet value invalid: %.2f", bet_value_float)
			fmt.Println(message)
			f.players[player_id].player.AddFailedUpdate(&game.UpdateMessage{Kind: "bet-failed",
				Content: gin.H{"message": message, "player_id": player_id}})
			return
		}
		bet_value := uint8(bet_value_float)
		if bet_value < 0 {
			message := fmt.Sprintf("Must bet at least 0 but bet %d", bet_value)
			fmt.Println(message)
			f.players[player_id].player.AddFailedUpdate(&game.UpdateMessage{Kind: "bet-failed",
				Content: gin.H{"message": message, "player_id": player_id}})
			return
		} else if bet_value > f.round {
			message := fmt.Sprintf("Cannot bet more than the cards in the round (%d) but bet %d", f.round, bet_value)
			fmt.Println(message)
			f.players[player_id].player.AddFailedUpdate(&game.UpdateMessage{Kind: "bet-failed",
				Content: gin.H{"message": message, "player_id": player_id}})
			return
		}
		f.players[f.turn].bet = bet_value
		done_betting := f.turn == f.dealer
		f.turn++
		if f.turn >= len(f.players) {
			f.turn -= len(f.players)
		}
		if done_betting {
			f.betting = false
			f.trick_leader = f.turn
		}
		game.Game_BroadcastUpdate(f, &game.UpdateMessage{Kind: "bet", Content: gin.H{
			"amount":    bet_value,
			"player_id": player_id,
		}})
	case "play-card":
		if player_id != f.turn {
			message := fmt.Sprintf("Not %d player's turn but %d player's turn", player_id, f.turn)
			fmt.Println(message)
			f.players[player_id].player.AddFailedUpdate(&game.UpdateMessage{Kind: "play-card-failed",
				Content: gin.H{"message": message, "player_id": player_id}})
			return
		}
		if f.betting {
			fmt.Println("Betting not playing cards")
			return
		}
		card_index_float, ok := action.Action["index"].(float64)
		if !ok {
			message := fmt.Sprintf("Card index invalid: %f", card_index_float)
			fmt.Println(message)
			f.players[player_id].player.AddFailedUpdate(&game.UpdateMessage{Kind: "play-card-failed",
				Content: gin.H{"message": message, "player_id": player_id}})
			return
		}
		card_index := int(card_index_float)
		if util.Contains(f.players[f.turn].cards_played, card_index) {
			message := fmt.Sprintf("Card with index %d already played", card_index)
			fmt.Println(message)
			f.players[player_id].player.AddFailedUpdate(&game.UpdateMessage{Kind: "play-card-failed",
				Content: gin.H{"message": message, "player_id": player_id}})
			return
		}
		cards := f.players[f.turn].cards
		if card_index < 0 || card_index >= len(cards) {
			message := fmt.Sprintf("Invalid card index %d for having %d cards", card_index, len(cards))
			fmt.Println(message)
			f.players[player_id].player.AddFailedUpdate(&game.UpdateMessage{Kind: "play-card-failed",
				Content: gin.H{"message": message, "player_id": player_id}})
			return
		}
		card := cards[card_index]
		if len(f.trick) > 0 {
			lead := f.trick[0]
			if lead != nil {
				suit := lead.GetSuit()
				if card.GetSuit() != suit {
					for i, other_card := range cards {
						if util.Contains(f.players[f.turn].cards_played, i) {
							continue
						}
						if other_card.GetSuit() == suit {
							message := fmt.Sprintf("Must follow suit of lead card %s and tried to play %s but have card that follows: %s",
								lead.GetName(), card.GetName(), other_card.GetName())
							fmt.Println(message)
							f.players[player_id].player.AddFailedUpdate(&game.UpdateMessage{Kind: "play-card-failed",
								Content: gin.H{"message": message, "player_id": player_id}})
							return
						}
					}
				}
			}
		}
		f.trick = append(f.trick, card)
		f.players[f.turn].cards_played = append(f.players[f.turn].cards_played, card_index)
		game.Game_BroadcastUpdate(f, &game.UpdateMessage{Kind: "play-card", Content: gin.H{
			"index":     card_index,
			"card":      card.ToFrontend(),
			"player_id": player_id,
		}})
		f.turn++
		if f.turn >= len(f.players) {
			f.turn -= len(f.players)
		}
		if f.turn == f.trick_leader {
			winning_index := 0
			winning_card := f.trick[0]
			for i, card := range f.trick[1:] {
				if card.GetSuit() == winning_card.GetSuit() {
					if card.GetNumber() > winning_card.GetNumber() {
						winning_index = i + 1
						winning_card = card
					}
				} else if card.GetSuit() == f.trump.GetSuit() {
					winning_index = i + 1
					winning_card = card
				}
			}
			f.turn = f.trick_leader + winning_index
			if f.turn >= len(f.players) {
				f.turn -= len(f.players)
			}
			f.players[f.turn].tricks++
			f.trick_leader = f.turn
			f.trick = []*game_utils.StandardCard{}
			fmt.Println("Trick won by", f.players[f.turn].player.GetNickname(), "with the", winning_card.GetName())
			if len(f.players[0].cards_played) < len(f.players[0].cards) {
				// next trick
			} else {
				for _, player := range f.players {
					if player.bet == player.tricks {
						player.score += f.round_points + f.trick_points*uint16(player.bet)
					}
				}
				f.dealNextRound()
			}
		}
	default:
		fmt.Println("Unknown game update type", action.Kind)
	}
}

func (f *GameFiddlesticks) PlayerDisconnected(client_id uint64) {
}

func (f *GameFiddlesticks) PlayerReconnected(client_id uint64) {
}

func (f *GameFiddlesticks) ToFrontend(client_id uint64, is_viewer bool) gin.H {
	game := gin.H{
		"round":             f.round,
		"max_round":         f.max_round,
		"rounds_increasing": f.rounds_increasing,
		"dealer":            f.dealer,
		"turn":              f.turn,
		"betting":           f.betting,
		"trick_leader":      f.trick_leader,
		"round_points":      f.round_points,
		"trick_points":      f.trick_points,
	}
	if f.game != nil {
		game["game_base"] = f.game.ToFrontend(client_id, is_viewer)
	}
	if f.deck != nil {
		game["deck"] = f.deck.ToFrontend()
	}
	players := []gin.H{}
	for _, player := range f.players {
		if player != nil {
			players = append(players, player.toFrontend(is_viewer || client_id == player.player.GetClientId()))
		}
	}
	game["players"] = players
	if f.trump != nil {
		game["trump"] = f.trump.ToFrontend()
	}
	trick_cards := []gin.H{}
	for _, card := range f.trick {
		if card != nil {
			trick_cards = append(trick_cards, card.ToFrontend())
		}
	}
	game["trick"] = trick_cards
	return game
}

func (f *GameFiddlesticks) dealNextRound() {
	if f.rounds_increasing && f.round == f.max_round {
		f.rounds_increasing = false
	}
	if f.round == 1 && !f.rounds_increasing {
		var winners = []int{0}
		var winning_score = f.players[0].score
		for i, player := range f.players[1:] {
			i++
			if player.score > winning_score {
				winning_score = player.score
				winners = []int{i}
			} else if player.score == winning_score {
				winners = append(winners, i)
			}
		}
		var winner_message = "The winner is: "
		if len(winners) > 1 {
			winner_message = "The winners are: "
		}
		for i, winner := range winners {
			winner_message += f.players[winner].player.GetNickname()
			if i+1 < len(winners) {
				winner_message += ", "
			}
		}
		winner_message += " with " + strconv.Itoa(int(winning_score)) + " points"
		fmt.Println(winner_message)
		f.game.EndGame()
		return
	}
	f.dealer++
	if f.dealer >= len(f.players) {
		f.dealer = 0
	}
	if f.rounds_increasing {
		f.round++
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
		f.players[j].cards_played = []int{}
		f.players[j].tricks = 0
		f.players[j].bet = 0
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
	f.trick_leader = f.turn
}
