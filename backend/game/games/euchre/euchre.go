package euchre

import (
	"github.com/dgray001/gray_online/game"
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
	game         *game.GameBase
	players      [4]*EuchrePlayer
	teams        [2]*EuchreTeam
	deck         *util.StandardDeck
	round        uint8
	dealer       int
	turn         int
	bidding      bool
	card_face_up *util.StandardCard
	trump_suit   uint8
	trick_leader int
	trick        []*util.StandardCard
}

type EuchrePlayer struct {
	player       *game.Player
	cards        []*util.StandardCard
	cards_played []int
}

type EuchreTeam struct {
	players [2]*EuchrePlayer
	score   uint8
	tricks  uint8
}

func (g *GameEuchre) GetId() uint64 {
	return g.game.Game_id
}

func (g *GameEuchre) GetBase() *game.GameBase {
	return g.game
}

func (g *GameEuchre) StartGame() {
	g.game.StartGame()
}

func (g *GameEuchre) Valid() bool {
	return false
}

func (g *GameEuchre) ToFrontend(client_id uint64, is_viewer bool) gin.H {
	return gin.H{}
}
