package euchre

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
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
	game         *game.GameBase
	players      [4]*EuchrePlayer
	teams        [2]*EuchreTeam
	deck         *game_utils.StandardDeck
	round        uint8
	dealer       int
	turn         int
	bidding      bool
	card_face_up *game_utils.StandardCard
	trump_suit   uint8
	trick_leader int
	trick        []*game_utils.StandardCard
}

func CreateGame(g *game.GameBase) (*GameEuchre, error) {
	euchre := GameEuchre{
		game:         g,
		players:      [4]*EuchrePlayer{},
		teams:        [2]*EuchreTeam{},
		deck:         game_utils.CreateStandardDeckConfig(game_utils.StandardDeckType_STANDARD24),
		round:        0,
		dealer:       -1,
		turn:         -1,
		bidding:      false,
		card_face_up: nil,
		trump_suit:   0,
		trick_leader: -1,
		trick:        []*game_utils.StandardCard{},
	}
	if len(g.Players) != 4 {
		//return nil, errors.New("Need exactly 4 players to play euchre")
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
		players: [2]*EuchrePlayer{euchre.players[0], euchre.players[2]},
		score:   0,
		tricks:  0,
	}
	euchre.teams[1] = &EuchreTeam{
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
	// TODO: implement
}

func (g *GameEuchre) Valid() bool {
	return false
}

func (g *GameEuchre) PlayerAction(action game.PlayerAction) {
	// TODO: implement
}

func (g *GameEuchre) PlayerDisconnected(client_id uint64) {
}

func (g *GameEuchre) PlayerReconnected(client_id uint64) {
}

func (g *GameEuchre) ToFrontend(client_id uint64, is_viewer bool) gin.H {
	game := gin.H{
		"round":        g.round,
		"dealer":       g.dealer,
		"turn":         g.turn,
		"bidding":      g.bidding,
		"trump_suit":   g.trump_suit,
		"trick_leader": g.trick_leader,
	}
	if g.game != nil {
		game["game_base"] = g.game.ToFrontend(client_id, is_viewer)
	}
	if g.deck != nil {
		game["deck"] = g.deck.ToFrontend()
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
