package fiddlesticks

import (
	"fmt"
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/gin-gonic/gin"
)

func (f *GameFiddlesticks) StartLearningDeal(hands [][]*game_utils.StandardCard, trump *game_utils.StandardCard, dealer int) error {
	if f.game == nil || f.game.GameStarted() || len(hands) != len(f.players) || len(hands) < 2 || dealer < 0 || dealer >= len(hands) || trump == nil || !trump.Valid() {
		return fmt.Errorf("invalid learning deal")
	}
	if err := validateLearningHands(hands, trump); err != nil {
		return err
	}
	hand := uint8(len(hands[0]))
	f.SetSettings(hand, hand, f.round_points, f.trick_points, 1)
	f.round, f.iteration, f.rounds_increasing, f.dealer, f.trump = hand, 0, true, dealer, trump
	f.turn = (dealer + 1) % len(hands)
	f.trick_leader = f.turn
	f.betting = true
	f.public_history, f.round_history, f.public_profiles = nil, nil, nil
	f.known_voids = make([][4]bool, len(hands))
	for seat, cards := range hands {
		p := f.players[seat]
		p.cards = append([]*game_utils.StandardCard(nil), cards...)
		p.cards_played = nil
		p.bet, p.tricks, p.score, p.has_bet = 0, 0, 0, false
	}
	f.game.StartGame()
	update := &game.UpdateMessage{Kind: "deal-round", Content: gin.H{"dealer": dealer, "round": hand, "trump": trump.ToFrontend()}}
	for _, p := range f.players {
		p.ai_model.ApplyUpdate(p, f, update)
	}
	return nil
}

func (f *GameFiddlesticks) LearningTurn() int { return f.turn }
