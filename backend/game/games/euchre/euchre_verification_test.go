package euchre

import (
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/gin-gonic/gin"
)

func setupTestGame(t *testing.T) (*GameEuchre, *game.GameBase) {
	base := game.CreateBaseGame(1, game.GameType_EUCHRE, nil, nil, nil)
	base.StartGame()
	for i := uint64(1); i <= 4; i++ {
		game.CreatePlayer(i, "Player", base)
	}
	g, err := CreateGame(base)
	if err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}
	return g, base
}

func TestGoingAloneDeadlockBackendExpectation(t *testing.T) {
	g, _ := setupTestGame(t)
	g.dealer = 0
	g.turn = 1
	g.bidding = true
	g.trick_leader = 1
	g.card_face_up = g.deck.DrawCard()

	// Player 1 bids and goes alone. Dealer is 0 (not player 1's partner, which is 3).
	p1Client := int(g.players[1].player.GetClientId())
	p1 := g.players[1].player
	g.PlayerAction(game.PlayerAction{
		Client_id: p1Client,
		Kind:      "bid",
		Action:    gin.H{"going_alone": true},
	})

	if !g.dealer_substituting_card {
		t.Fatalf("Expected dealer_substituting_card to be true because player 1 is not dealer's partner")
	}

	// Player 1 tries to play a card because frontend skips substitution and calls setPlaying()
	g.PlayerAction(game.PlayerAction{
		Client_id: p1Client,
		Kind:      "play-card",
		Action:    gin.H{"index": float64(0)},
	})

	select {
	case failed := <-p1.FailedUpdates:
		if failed.Kind != "play-card-failed" {
			t.Fatalf("Expected play-card-failed, got %s", failed.Kind)
		}
		if failed.Content["message"] != "Not currently playing card" {
			t.Fatalf("Unexpected message: %v", failed.Content["message"])
		}
	default:
		t.Fatalf("Expected play-card to be rejected with play-card-failed")
	}
}

func TestGoingAlonePartnerBypassesSubstitution(t *testing.T) {
	g, _ := setupTestGame(t)
	g.dealer = 0
	g.turn = 2
	g.bidding = true
	g.trick_leader = 1
	g.card_face_up = g.deck.DrawCard()

	// Player 2 is dealer 0's partner ((2+2)%4 == 0). Bids and goes alone.
	p2Client := int(g.players[2].player.GetClientId())
	g.PlayerAction(game.PlayerAction{
		Client_id: p2Client,
		Kind:      "bid",
		Action:    gin.H{"going_alone": true},
	})

	// Dealer sits out, so dealer substitution should be false
	if g.dealer_substituting_card {
		t.Fatalf("Expected dealer_substituting_card to be false because bidder is dealer's partner")
	}
}

func TestDealerPlayOutOfTurnRejected(t *testing.T) {
	g, _ := setupTestGame(t)
	g.dealer = 0
	g.turn = 1
	g.bidding = true
	g.trick_leader = 1
	g.card_face_up = g.deck.DrawCard()

	// Deal 5 cards to dealer
	for i := 0; i < 5; i++ {
		g.players[0].cards = append(g.players[0].cards, g.deck.DrawCard())
	}

	// Player 1 bids (normal bid)
	p1Client := int(g.players[1].player.GetClientId())
	g.PlayerAction(game.PlayerAction{
		Client_id: p1Client,
		Kind:      "bid",
		Action:    gin.H{"going_alone": false},
	})

	// Dealer substitutes card
	p0Client := int(g.players[0].player.GetClientId())
	p0 := g.players[0].player
	g.PlayerAction(game.PlayerAction{
		Client_id: p0Client,
		Kind:      "dealer-substitutes-card",
		Action:    gin.H{"index": float64(0)},
	})

	if g.dealer_substituting_card {
		t.Fatalf("Expected dealer_substituting_card to be false after substitution")
	}

	// Dealer tries to play out of turn (turn is 1)
	g.PlayerAction(game.PlayerAction{
		Client_id: p0Client,
		Kind:      "play-card",
		Action:    gin.H{"index": float64(0)},
	})

	select {
	case failed := <-p0.FailedUpdates:
		if failed.Kind != "play-card-failed" {
			t.Fatalf("Expected play-card-failed, got %s", failed.Kind)
		}
	default:
		t.Fatalf("Expected dealer play-card to be rejected out of turn")
	}
}
