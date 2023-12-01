package euchre

import (
	"errors"
	"fmt"
	"os"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

/*
   ==================
   >>>>> EUCHRE <<<<<
	 ==================

	 Objective: first team to 10 points wins
	 Description: using a 24-card deck (9s to aces), 2 teams of 2 get dealt 5
	   cards per hand with one card at the end dealt face up. Generally, the team
		 that wins the most tricks will win the hand and score points. After each
		 hand, the dealer will switch to the person to the left of the last dealer.
	 Bidding: starting to the left of dealer, each player has a chance to bid or
	   pass. If a player bids, they tell the dealer to 'pick it up' and the dealer
		 trades one of their cards with the card dealt face up. The suit of the card
		 dealt face is the trump for the hand. The team of the player that bid is
		 the making team, while the other team is the defending team. The player
		 that bids can also choose to go alone, in which case their teammate cannot
		 play for that round. If no player bids, then the card dealt face up is
		 flipped over and each player then has a chance to bid by choosing a trump
		 not of the card that was dealt face up. If no one chooses a trump and the
		 bid goes to the dealer for the second time, the dealer must choose a trump
		 and their team must be the makers.
	 Tricks: play starts to left of dealer, players must follow suit, and the
	   winner of a trick starts the next trick.
	 Trump: the jack of trump is the high trump, and the jack of the matching-color
	   suit is the second-highest trump. Trump then goes from ace to 9. Non-trump
		 suits act normally with the ace as the highest.
	 Scoring: If the making team wins 3 or 4 tricks they score 1 point, if they
	   win all 5 tricks, they score 2 points, if they score less than 3 points the
		 defenders score 2 points. If the maker went alone they score 4 points if
		 they win all 5 tricks.
*/

type GameEuchre struct {
	game                     *game.GameBase
	players                  [4]*EuchrePlayer
	teams                    [2]*EuchreTeam
	deck                     *game_utils.StandardDeck
	round                    uint8
	dealer                   int
	turn                     int
	bidding                  bool
	bidding_choose_trump     bool
	dealer_substituting_card bool
	player_bid               int
	makers_team              int
	defenders_team           int
	going_alone              bool
	card_face_up             *game_utils.StandardCard
	trump_suit               uint8
	trick_leader             int
	trick                    []*game_utils.StandardCard
	trick_number             uint8
}

func CreateGame(g *game.GameBase) (*GameEuchre, error) {
	euchre := GameEuchre{
		game:                     g,
		players:                  [4]*EuchrePlayer{},
		teams:                    [2]*EuchreTeam{},
		deck:                     game_utils.CreateStandardDeckConfig(game_utils.StandardDeckType_STANDARD24),
		round:                    0,
		dealer:                   -1,
		turn:                     -1,
		bidding:                  false,
		bidding_choose_trump:     false,
		dealer_substituting_card: false,
		player_bid:               -1,
		makers_team:              -1,
		defenders_team:           -1,
		going_alone:              false,
		card_face_up:             nil,
		trump_suit:               0,
		trick_leader:             -1,
		trick:                    []*game_utils.StandardCard{},
		trick_number:             1,
	}
	if len(g.Players) != 4 {
		return nil, errors.New("Need exactly 4 players to play euchre")
	}
	player_id := 0
	for _, player := range g.Players {
		player.Player_id = player_id
		euchre.players[player_id] = &EuchrePlayer{
			player:       player,
			cards:        []*game_utils.StandardCard{},
			cards_played: []int{},
		}
		player_id++
	}
	euchre.teams[0] = &EuchreTeam{
		team_id: 0,
		players: [2]*EuchrePlayer{euchre.players[0], euchre.players[2]},
		score:   0,
		tricks:  0,
	}
	euchre.teams[1] = &EuchreTeam{
		team_id: 1,
		players: [2]*EuchrePlayer{euchre.players[1], euchre.players[3]},
		score:   0,
		tricks:  0,
	}
	return &euchre, nil
}

func (g *GameEuchre) GetBase() *game.GameBase {
	return g.game
}

func (g *GameEuchre) StartGame() {
	g.dealer = util.RandomInt(0, len(g.players)-1)
	g.dealNextRound()
}

func (g *GameEuchre) Valid() bool {
	if g.game == nil || g.deck == nil {
		return false
	}
	// TODO: implement
	return true
}

func (g *GameEuchre) PlayerAction(action game.PlayerAction) {
	fmt.Println("player action:", action.Kind, action.Client_id, action.Action)
	player := g.game.Players[uint64(action.Client_id)]
	if player == nil {
		fmt.Fprintln(os.Stderr, "Invalid client id", action.Client_id)
		return
	}
	player_id := player.Player_id
	switch action.Kind {
	case "pass":
		if !g.bidding && !g.bidding_choose_trump {
			player.AddFailedUpdateShorthand("pass-failed", "Not currently bidding")
			return
		}
		if player_id != g.turn {
			player.AddFailedUpdateShorthand("bid-failed",
				fmt.Sprintf("Not %d player's turn but %d player's turn", player_id, g.turn))
			return
		}
		if g.bidding_choose_trump && g.turn == g.dealer {
			player.AddFailedUpdateShorthand("pass-failed", "Dealer is stuck and must choose trump")
			return
		}
		g.executePass(player)
	case "bid":
		if !g.bidding {
			player.AddFailedUpdateShorthand("bid-failed", "Not currently bidding")
			return
		}
		if player_id != g.turn {
			player.AddFailedUpdateShorthand("bid-failed",
				fmt.Sprintf("Not %d player's turn but %d player's turn", player_id, g.turn))
			return
		}
		going_alone, ok := action.Action["going_alone"].(bool)
		if !ok {
			player.AddFailedUpdateShorthand("bid-failed",
				fmt.Sprintf("Going alone invalid: %t", going_alone))
			return
		}
		g.executeBid(player, going_alone)
	case "bid-choose-trump":
		if !g.bidding_choose_trump {
			player.AddFailedUpdateShorthand("bid-choose-trump-failed", "Not currently bidding and choosing trump")
			return
		}
		if player_id != g.turn {
			player.AddFailedUpdateShorthand("bid-choose-trump-failed",
				fmt.Sprintf("Not %d player's turn but %d player's turn", player_id, g.turn))
			return
		}
		going_alone, ok := action.Action["going_alone"].(bool)
		if !ok {
			player.AddFailedUpdateShorthand("bid-choose-trump-failed",
				fmt.Sprintf("Going alone invalid: %t", going_alone))
			return
		}
		trump_suit_float, ok := action.Action["trump_suit"].(float64)
		if !ok {
			player.AddFailedUpdateShorthand("bid-choose-trump-failed",
				fmt.Sprintf("Trump suit value invalid: %t", going_alone))
			return
		}
		trump_suit := uint8(trump_suit_float)
		if trump_suit < 1 || trump_suit > 4 {
			player.AddFailedUpdateShorthand("bid-choose-trump-failed",
				fmt.Sprintf("Not a valid suit number: %t", going_alone))
			return
		}
	case "dealer-substitutes-card":
		if !g.dealer_substituting_card {
			player.AddFailedUpdateShorthand("dealer-substitutes-card-failed", "Dealer not currently substituting card")
			return
		}
		if player_id != g.dealer {
			player.AddFailedUpdateShorthand("dealer-substitutes-card-failed", "Only dealer can substitute card")
			return
		}
		card_index_float, ok := action.Action["index"].(float64)
		if !ok {
			player.AddFailedUpdateShorthand("dealer-substitutes-card-failed",
				fmt.Sprintf("Card substituting index value invalid: %f", card_index_float))
			return
		}
		card_index := int(card_index_float)
		cards := g.players[g.dealer].cards
		if card_index < 0 || card_index >= len(cards) {
			player.AddFailedUpdateShorthand("dealer-substitutes-card-failed",
				fmt.Sprintf("Invalid card index %d for having %d cards", card_index, len(cards)))
			return
		}
		g.executeDealerSubstitutesCard(player, card_index)
	case "play-card":
		if g.bidding || g.bidding_choose_trump || g.dealer_substituting_card {
			player.AddFailedUpdateShorthand("play-card-failed", "Not currently playing card")
			return
		}
		if player_id != g.turn {
			player.AddFailedUpdateShorthand("play-card-failed",
				fmt.Sprintf("Not %d player's turn but %d player's turn", player_id, g.turn))
			return
		}
		card_index_float, ok := action.Action["index"].(float64)
		if !ok {
			player.AddFailedUpdateShorthand("play-card-failed",
				fmt.Sprintf("Card index invalid: %f", card_index_float))
			return
		}
		card_index := int(card_index_float)
		if util.Contains(g.players[g.turn].cards_played, card_index) {
			player.AddFailedUpdateShorthand("play-card-failed",
				fmt.Sprintf("Card with index %d already played", card_index))
			return
		}
		cards := g.players[g.turn].cards
		if card_index < 0 || card_index >= len(cards) {
			player.AddFailedUpdateShorthand("play-card-failed",
				fmt.Sprintf("Invalid card index %d for having %d cards", card_index, len(cards)))
			return
		}
		card := cards[card_index]
		if len(g.trick) > 0 {
			lead := g.trick[0]
			if lead != nil {
				suit := lead.GetSuit()
				if card.GetSuit() != suit {
					for i, other_card := range cards {
						if util.Contains(g.players[g.turn].cards_played, i) {
							continue
						}
						if other_card.GetSuit() == suit {
							player.AddFailedUpdateShorthand("play-card-failed",
								fmt.Sprintf("Must follow suit of lead card %s and tried to play %s but have card that follows: %s",
									lead.GetName(), card.GetName(), other_card.GetName()))
							return
						}
					}
				}
			}
		}
		g.executePlayCard(player, card_index)
	default:
		fmt.Fprintln(os.Stderr, "Unknown game update type", action.Kind)
	}
}

func (g *GameEuchre) executePass(player *game.Player) {
	if g.turn == g.dealer {
		g.bidding = false
		g.bidding_choose_trump = true
	}
	g.turn++
	g.resolveTurn()
	game.Game_BroadcastUpdate(g, &game.UpdateMessage{Kind: "pass", Content: gin.H{
		"player_id": player.Player_id,
	}})
}

func (g *GameEuchre) executeBid(player *game.Player, going_alone bool) {
	g.bidding = false
	g.makers_team = player.Player_id % 2
	g.defenders_team = 0
	if g.makers_team == 0 {
		g.defenders_team = 1
	}
	g.player_bid = player.Player_id
	g.going_alone = going_alone
	g.turn = g.dealer + 1
	g.resolveTurn()
	g.trick_leader = g.turn
	g.trump_suit = g.card_face_up.GetSuit()
	g.dealer_substituting_card = true
	game.Game_BroadcastUpdate(g, &game.UpdateMessage{Kind: "bid", Content: gin.H{
		"player_id":   player.Player_id,
		"going_alone": going_alone,
	}})
}

func (g *GameEuchre) executeBidChooseTrump(player *game.Player, going_alone bool, trump_suit uint8) {
	g.bidding_choose_trump = false
	g.makers_team = player.Player_id % 2
	g.defenders_team = 0
	if g.makers_team == 0 {
		g.defenders_team = 1
	}
	g.player_bid = player.Player_id
	g.going_alone = going_alone
	g.turn = g.dealer + 1
	g.resolveTurn()
	g.trick_leader = g.turn
	g.trump_suit = trump_suit
	game.Game_BroadcastUpdate(g, &game.UpdateMessage{Kind: "bid-choose-trump", Content: gin.H{
		"player_id":   player.Player_id,
		"going_alone": going_alone,
		"trump_suit":  trump_suit,
	}})
}

func (g *GameEuchre) executeDealerSubstitutesCard(player *game.Player, card_index int) {
	g.dealer_substituting_card = false
	g.players[g.dealer].cards[card_index] = g.card_face_up
	game.Game_BroadcastUpdate(g, &game.UpdateMessage{Kind: "dealer-substitutes-card", Content: gin.H{
		"player_id":  player.Player_id,
		"card_index": card_index,
	}})
}

func (g *GameEuchre) executePlayCard(player *game.Player, card_index int) {
	card := g.players[g.turn].cards[card_index]
	g.trick = append(g.trick, card)
	g.players[g.turn].cards_played = append(g.players[g.turn].cards_played, card_index)
	game.Game_BroadcastUpdate(g, &game.UpdateMessage{Kind: "play-card", Content: gin.H{
		"index":     card_index,
		"card":      card.ToFrontend(),
		"player_id": player.Player_id,
	}})
	g.turn++
	g.resolveTurn()
	if g.turn != g.trick_leader {
		return
	}
	// end of trick
	winning_index := 0
	winning_card := g.trick[0]
	for i, card := range g.trick[1:] {
		if g.cardSuit(card) == g.cardSuit(winning_card) {
			if g.cardNumber(card) > g.cardNumber(winning_card) {
				winning_index = i + 1
				winning_card = card
			}
		} else if g.cardSuit(card) == g.trump_suit {
			winning_index = i + 1
			winning_card = card
		}
	}
	g.turn = g.trick_leader + winning_index
	if g.turn >= len(g.players) {
		g.turn -= len(g.players)
	}
	g.teams[g.players[g.turn].player.Player_id%2].tricks++
	g.trick_leader = g.turn
	g.trick = []*game_utils.StandardCard{}
	fmt.Println("Trick won by", g.players[g.turn].player.GetNickname(), "with the", winning_card.GetName())
	if g.trick_number < 5 {
		g.trick_number++
		return
	}
	// end of round
	winning_team := 0
	if g.teams[0].tricks < 3 {
		winning_team = 1
	}
	won_all_five := g.teams[winning_team].tricks == 5
	if winning_team == g.makers_team {
		// makers won
		if won_all_five {
			if g.going_alone {
				g.scorePoints(g.makers_team, 4)
			} else {
				g.scorePoints(g.makers_team, 2)
			}
		} else {
			g.scorePoints(g.makers_team, 1)
		}
	} else {
		// defenders won
		g.scorePoints(g.defenders_team, 2)
	}
	if !g.game.GameEnded() {
		g.dealNextRound()
	}
}

func (g *GameEuchre) resolveTurn() {
	if g.turn >= len(g.players) {
		g.turn -= len(g.players)
	}
	if g.going_alone && util.AbsDiffInt(g.turn, g.player_bid) == 2 {
		g.turn++
		if g.turn >= len(g.players) {
			g.turn -= len(g.players)
		}
	}
}

func (g *GameEuchre) scorePoints(team int, points uint8) {
	g.teams[team].score += points
	if g.teams[team].score >= 10 {
		winning_message := fmt.Sprintf("The team of %s and %s won the game with %d points",
			g.teams[team].players[0].player.GetNickname(), g.teams[team].players[1].player.GetNickname(),
			g.teams[team].score)
		fmt.Println(winning_message)
		g.game.EndGame(winning_message)
	}
}

func (g *GameEuchre) cardSuit(card *game_utils.StandardCard) uint8 {
	if card.GetSuitColor() == game_utils.SuitColor(g.trump_suit) && card.GetNumber() == 11 {
		return g.trump_suit
	}
	return card.GetSuit()
}

func (g *GameEuchre) cardNumber(card *game_utils.StandardCard) uint8 {
	if card.GetSuit() == g.trump_suit && card.GetNumber() == 11 {
		// right bar
		return 16
	} else if g.cardSuit(card) == g.trump_suit && card.GetNumber() == 11 {
		// left bar
		return 15
	}
	return card.GetNumber()
}

func (g *GameEuchre) PlayerDisconnected(client_id uint64) {
}

func (g *GameEuchre) PlayerReconnected(client_id uint64) {
}

func (g *GameEuchre) ToFrontend(client_id uint64, is_viewer bool) gin.H {
	game := gin.H{
		"round":                    g.round,
		"dealer":                   g.dealer,
		"turn":                     g.turn,
		"bidding":                  g.bidding,
		"bidding_choose_trump":     g.bidding_choose_trump,
		"player_bid":               g.player_bid,
		"makers_team":              g.makers_team,
		"defenders_team":           g.defenders_team,
		"going_alone":              g.going_alone,
		"dealer_substituting_card": g.dealer_substituting_card,
		"trump_suit":               g.trump_suit,
		"trick_leader":             g.trick_leader,
	}
	if g.game != nil {
		game["game_base"] = g.game.ToFrontend(client_id, is_viewer)
	}
	players := []gin.H{}
	for _, player := range g.players {
		if player != nil {
			players = append(players, player.toFrontend(is_viewer || client_id == player.player.GetClientId()))
		}
	}
	game["players"] = players
	teams := []gin.H{}
	for _, team := range g.teams {
		if team != nil {
			teams = append(teams, team.toFrontend())
		}
	}
	game["teams"] = teams
	if g.card_face_up != nil {
		game["card_face_up"] = g.card_face_up.ToFrontend()
	}
	trick_cards := []gin.H{}
	for _, card := range g.trick {
		if card != nil {
			trick_cards = append(trick_cards, card.ToFrontend())
		}
	}
	game["trick"] = trick_cards
	return game
}

func (g *GameEuchre) dealNextRound() {
	g.dealer++
	if g.dealer >= len(g.players) {
		g.dealer = 0
	}
	g.round++
	g.deck.Reset()
	dealt_cards := g.deck.DealCards(uint8(len(g.players)), 5)
	for i := 0; i < len(g.players); i++ {
		cards := dealt_cards[i]
		j := i + g.dealer
		if j >= len(g.players) {
			j -= len(g.players)
		}
		g.players[j].cards = cards
		g.players[j].cards_played = []int{}
	}
	for i := 0; i < len(g.teams); i++ {
		g.teams[i].tricks = 0
	}
	g.card_face_up = g.deck.DrawCard()
	for _, player := range g.players {
		frontend_cards := []gin.H{}
		for _, card := range player.cards {
			frontend_cards = append(frontend_cards, card.ToFrontend())
		}
		player.player.AddUpdate(&game.UpdateMessage{Kind: "deal-round", Content: gin.H{
			"dealer":       g.dealer,
			"round":        g.round,
			"card_face_up": g.card_face_up.ToFrontend(),
			"cards":        frontend_cards,
		}})
	}
	g.game.AddViewerUpdate(&game.UpdateMessage{Kind: "deal-round", Content: gin.H{
		"dealer":       g.dealer,
		"round":        g.round,
		"card_face_up": g.card_face_up.ToFrontend(),
	}})
	g.bidding = true
	g.turn = g.dealer + 1
	if g.turn >= len(g.players) {
		g.turn -= len(g.players)
	}
	g.trick_leader = g.turn
}
