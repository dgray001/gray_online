package fiddlesticks

import (
	"errors"
	"fmt"
	"os"
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
	min_round         uint8
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

type FiddlesticksAiPlayerFromFrontend struct {
	nickname string
}

func InitializeGame(g *game.GameBase) *GameFiddlesticks {
	return &GameFiddlesticks{
		game:              g,
		players:           []*FiddlesticksPlayer{},
		deck:              game_utils.CreateStandardDeck(),
		round:             0,
		min_round:         1,
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
}

func CreateGame(g *game.GameBase, action_channel chan game.PlayerAction) (*GameFiddlesticks, error) {
	fiddlesticks := InitializeGame(g)
	var player_id = 0
	for _, player := range g.Players {
		player.Player_id = player_id
		fiddlesticks.players = append(fiddlesticks.players, &FiddlesticksPlayer{
			player:       player,
			cards:        []*game_utils.StandardCard{},
			cards_played: []int{},
			score:        0,
			ai_model_id:  1,
		})
		player_id++
	}
	ai_players, ai_players_ok := g.GameSpecificSettings["ai_players"].([]interface{})
	if ai_players_ok {
		for _, ai_player := range ai_players {
			ai, ai_ok := ai_player.(map[string]interface{})
			if !ai_ok {
				fmt.Println("Failed to cast ai: ", ai_player)
				continue
			}
			nickname, nickname_ok := ai["nickname"].(string)
			if !nickname_ok {
				fmt.Println("Failed to cast nickname: ", ai["nickname"])
				continue
			}
			player := game.CreateAiPlayer(nickname, g)
			fiddlesticks_player := &FiddlesticksPlayer{
				player:       player,
				cards:        []*game_utils.StandardCard{},
				cards_played: []int{},
				score:        0,
				ai_model_id:  1,
			}
			go runAi(fiddlesticks_player, fiddlesticks, action_channel)
			player.Player_id = player_id
			fiddlesticks.players = append(fiddlesticks.players, fiddlesticks_player)
			player_id++
		}
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
	min_round_float, min_round_ok := g.GameSpecificSettings["min_round"].(float64)
	if min_round_ok {
		min_round := uint8(min_round_float)
		if min_round > 0 && min_round <= fiddlesticks.max_round {
			fiddlesticks.min_round = min_round
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
	return fiddlesticks, nil
}

func (f *GameFiddlesticks) AddInternalAiPlayer(ai_model_id uint8) {
	player := &game.Player{
		Player_id: len(f.players),
	}
	fiddlesticks_player := &FiddlesticksPlayer{
		player:       player,
		cards:        []*game_utils.StandardCard{},
		cards_played: []int{},
		score:        0,
		ai_model_id:  ai_model_id,
	}
	fiddlesticks_player.createAiModel()
	f.players = append(f.players, fiddlesticks_player)
}

func (f *GameFiddlesticks) SetSettings(min_round uint8, max_round uint8, round_points uint16, trick_points uint16) {
	f.min_round = min_round
	f.max_round = max_round
	f.round_points = round_points
	f.trick_points = trick_points
}

func (f *GameFiddlesticks) GetBase() *game.GameBase {
	return f.game
}

func (f *GameFiddlesticks) StartGame() {
	f.round = f.min_round - 1
	f.dealer = util.RandomInt(0, len(f.players)-1)
	f.dealNextRound(true)
}

func (f *GameFiddlesticks) StartAiGame() {
	f.round = f.min_round - 1
	f.dealer = len(f.players) - 1
	f.dealNextRound(false)
}

/** Returns whether the game is over */
func (f *GameFiddlesticks) ExecuteAiTurn() bool {
	if !f.GetBase().GameStarted() {
		return false
	}
	if f.GetBase().GameEnded() {
		return true
	}
	player := f.players[f.turn]
	if f.betting {
		bid := GetAiBid(player, f)
		fmt.Println("AI player", f.turn, "betting", bid)
		f.executeBet(player.player, bid, false)
	} else {
		card_index := GetAiPlayCard(player, f)
		fmt.Println("AI player", f.turn, "playing", player.cards[card_index].GetName())
		f.executePlayCard(player.player, card_index, false)
	}
	return false
}

func (f *GameFiddlesticks) Valid() bool {
	if f.game == nil || f.deck == nil {
		return false
	}
	return true
}

func (f *GameFiddlesticks) PlayerAction(action game.PlayerAction) {
	fmt.Println("Player action:", action.Kind, action.Client_id, action.Ai_id, action.Action)
	player := f.game.AiPlayers[uint32(action.Ai_id)]
	if player == nil {
		player = f.game.Players[uint64(action.Client_id)]
	}
	if player == nil {
		fmt.Fprintln(os.Stderr, "Invalid client or ai id", action.Client_id, action.Ai_id)
		return
	}
	player_id := player.Player_id
	switch action.Kind {
	case "bet":
		if !f.betting {
			player.AddFailedUpdateShorthand("bet-failed", "Not currently betting")
			return
		}
		if player_id != f.turn {
			player.AddFailedUpdateShorthand("bet-failed",
				fmt.Sprintf("Not %d player's turn but %d player's turn", player_id, f.turn))
			return
		}
		bet_value_float, ok := action.Action["amount"].(float64)
		if !ok {
			player.AddFailedUpdateShorthand("bet-failed", fmt.Sprintf("Bet value invalid: %.2f", bet_value_float))
			return
		}
		bet_value := uint8(bet_value_float)
		if bet_value < 0 {
			player.AddFailedUpdateShorthand("bet-failed", fmt.Sprintf("Must bet at least 0 but bet %d", bet_value))
			return
		} else if bet_value > f.round {
			player.AddFailedUpdateShorthand("bet-failed",
				fmt.Sprintf("Cannot bet more than the cards in the round (%d) but bet %d", f.round, bet_value))
			return
		}
		f.executeBet(player, bet_value, true)
	case "play-card":
		if f.betting {
			player.AddFailedUpdateShorthand("play-card-failed", "Betting not playing cards")
			return
		}
		if player_id != f.turn {
			player.AddFailedUpdateShorthand("play-card-failed",
				fmt.Sprintf("Not %d player's turn but %d player's turn", player_id, f.turn))
			return
		}
		card_index_float, ok := action.Action["index"].(float64)
		if !ok {
			player.AddFailedUpdateShorthand("play-card-failed",
				fmt.Sprintf("Card index invalid: %f", card_index_float))
			return
		}
		card_index := int(card_index_float)
		if util.Contains(f.players[f.turn].cards_played, card_index) {
			player.AddFailedUpdateShorthand("play-card-failed",
				fmt.Sprintf("Card with index %d already played", card_index))
			return
		}
		cards := f.players[f.turn].cards
		if card_index < 0 || card_index >= len(cards) {
			player.AddFailedUpdateShorthand("play-card-failed",
				fmt.Sprintf("Invalid card index %d for having %d cards", card_index, len(cards)))
			return
		}
		card := cards[card_index]
		valid_cards, lead_suit := f.validCards(player_id)
		if !util.Contains(valid_cards, card_index) {
			player.AddFailedUpdateShorthand("play-card-failed",
				fmt.Sprintf("Must follow suit of lead card %d but tried to play %s", lead_suit, card.GetName()))
			return
		}
		f.executePlayCard(player, card_index, true)
	default:
		fmt.Fprintln(os.Stderr, "Unknown game update type", action.Kind)
	}
}

func (f *GameFiddlesticks) validCards(player_id int) ([]int, uint8) {
	cards := f.players[player_id].cards
	cards_played := f.players[player_id].cards_played
	valid_cards := []int{}
	lead_suit := uint8(0)
	has_lead_suit := false
	if len(f.trick) > 0 && f.trick[0] != nil {
		lead_suit = f.trick[0].GetSuit()
		for card_id, card := range cards {
			if !util.Contains(cards_played, card_id) && lead_suit == card.GetSuit() {
				has_lead_suit = true
				break
			}
		}
	}
	for card_id, card := range cards {
		if util.Contains(cards_played, card_id) {
			continue
		}
		if has_lead_suit && lead_suit != card.GetSuit() {
			continue
		}
		valid_cards = append(valid_cards, card_id)
	}
	return valid_cards, lead_suit
}

func (f *GameFiddlesticks) winningTrickCard() (*game_utils.StandardCard, int) {
	if len(f.trick) == 0 {
		return nil, -1
	}
	winning_index := 0
	winning_card := f.trick[0]
	for i, card := range f.trick[1:] {
		if f.cardBeatsCard(card, winning_card) {
			winning_index = i + 1
			winning_card = card
		}
	}
	return winning_card, winning_index
}

func (f *GameFiddlesticks) cardBeatsCard(c1 *game_utils.StandardCard, c2 *game_utils.StandardCard) bool {
	if c1 == nil {
		return false
	}
	if c2 == nil {
		return true
	}
	if c1.GetSuit() != c2.GetSuit() {
		return c1.GetSuit() == f.trump.GetSuit()
	}
	return c1.GetNumber() > c2.GetNumber()
}

func (f *GameFiddlesticks) executeBet(player *game.Player, bet_value uint8, broadcast bool) {
	f.players[f.turn].bet = bet_value
	f.players[f.turn].has_bet = true
	done_betting := f.turn == f.dealer
	f.turn++
	if f.turn >= len(f.players) {
		f.turn -= len(f.players)
	}
	if done_betting {
		f.betting = false
		f.trick_leader = f.turn
	}
	if broadcast {
		update := &game.UpdateMessage{Kind: "bet", Content: gin.H{
			"amount":    bet_value,
			"player_id": player.Player_id,
		}}
		game.Game_BroadcastUpdate(f, update)
	}
}

func (f *GameFiddlesticks) executePlayCard(player *game.Player, card_index int, broadcast bool) {
	card := f.players[f.turn].cards[card_index]
	f.trick = append(f.trick, card)
	f.players[f.turn].cards_played = append(f.players[f.turn].cards_played, card_index)
	f.turn++
	if f.turn >= len(f.players) {
		f.turn -= len(f.players)
	}
	deal_next_round := false
	if f.turn == f.trick_leader {
		winning_card, winning_index := f.winningTrickCard()
		f.turn = f.trick_leader + winning_index
		if f.turn >= len(f.players) {
			f.turn -= len(f.players)
		}
		f.players[f.turn].tricks++
		f.trick_leader = f.turn
		f.trick = []*game_utils.StandardCard{}
		if broadcast {
			fmt.Println("Trick won by", f.players[f.turn].player.GetNickname(), "with the", winning_card.GetName())
		}
		if len(f.players[0].cards_played) < len(f.players[0].cards) {
			// next trick
		} else {
			for _, player := range f.players {
				if player.bet == player.tricks {
					player.score += f.round_points + f.trick_points*uint16(player.bet)
				}
			}
			deal_next_round = true
		}
	}
	update := &game.UpdateMessage{Kind: "play-card", Content: gin.H{
		"index":     card_index,
		"card":      card.ToFrontend(),
		"player_id": player.Player_id,
	}}
	if broadcast {
		game.Game_BroadcastUpdate(f, update)
	}
	if deal_next_round {
		f.dealNextRound(broadcast)
	}
}

func (f *GameFiddlesticks) PlayerDisconnected(client_id uint64) {
}

func (f *GameFiddlesticks) PlayerReconnected(client_id uint64) {
}

func (f *GameFiddlesticks) ToFrontend(client_id uint64, is_viewer bool) gin.H {
	game := gin.H{
		"round":             f.round,
		"min_round":         f.min_round,
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

func (f *GameFiddlesticks) dealNextRound(broadcast bool) {
	if f.rounds_increasing && f.round == f.max_round {
		f.rounds_increasing = false
	}
	if f.round == f.min_round && !f.rounds_increasing {
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
		if broadcast {
			fmt.Println(winner_message)
		}
		f.game.EndGame(winner_message)
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
		f.players[j].has_bet = false
	}
	f.trump = f.deck.DrawCard()
	f.betting = true
	f.turn = f.dealer + 1
	if f.turn >= len(f.players) {
		f.turn -= len(f.players)
	}
	f.trick_leader = f.turn
	for _, player := range f.players {
		frontend_cards := []gin.H{}
		for _, card := range player.cards {
			frontend_cards = append(frontend_cards, card.ToFrontend())
		}
		update := &game.UpdateMessage{Kind: "deal-round", Content: gin.H{
			"dealer": f.dealer,
			"round":  f.round,
			"trump":  f.trump.ToFrontend(),
			"cards":  frontend_cards,
		}}
		if broadcast {
			player.player.AddUpdate(update)
		}
	}
	if broadcast {
		f.game.AddViewerUpdate(&game.UpdateMessage{Kind: "deal-round", Content: gin.H{
			"dealer": f.dealer,
			"round":  f.round,
			"trump":  f.trump.ToFrontend(),
		}})
	}
}
