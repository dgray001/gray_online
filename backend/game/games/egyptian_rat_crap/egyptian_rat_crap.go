package egyptian_rat_crap

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

/*
   ===============================
   >>>>> EGYPTIAN RAT CRAP <<<<<
   ===============================

   Objective: be the last player left with cards, by collecting the whole deck
   Description: the deck is dealt evenly into a face-down pile for each player, unseen
     by anyone including its owner. On their turn a player flips the top card of their
     own pile onto a shared central pile. Flipping a face card or ace challenges the next
     player to flip a counter face card/ace within a limited number of flips (jack: 1,
     queen: 2, king: 3, ace: 4); if they fail, whoever played the most recent unanswered
     face card/ace takes the whole central pile onto the bottom of their own pile. A
     player whose pile runs out is eliminated.
*/

type GameEgyptianRatCrap struct {
	game             *game.GameBase
	players          []*EgyptianRatCrapPlayer
	central_pile     []*game_utils.StandardCard
	turn             int
	challenge_active bool
	challenger_id    int
	chances_left     uint8
	// nil unless a resolution grace window is currently pending
	challenge_timer     *time.Timer
	turn_announce_timer *time.Timer
	turn_timer          *time.Timer
	pending_turn_winner int
	action_channel      chan game.PlayerAction
}

func CreateGame(g *game.GameBase, action_channel chan game.PlayerAction) (*GameEgyptianRatCrap, error) {
	egyptian_rat_crap := GameEgyptianRatCrap{
		game:           g,
		players:        []*EgyptianRatCrapPlayer{},
		turn:           -1,
		action_channel: action_channel,
	}
	var player_id = 0
	for _, player := range g.Players {
		player.Player_id = player_id
		egyptian_rat_crap.players = append(egyptian_rat_crap.players, &EgyptianRatCrapPlayer{
			player: player,
			pile:   []*game_utils.StandardCard{},
		})
		player_id++
	}
	if len(egyptian_rat_crap.players) < 2 {
		return nil, errors.New("need at least two players to play egyptian rat crap")
	}
	return &egyptian_rat_crap, nil
}

func (g *GameEgyptianRatCrap) GetBase() *game.GameBase {
	return g.game
}

func (g *GameEgyptianRatCrap) StartGame() {
	deck := game_utils.CreateStandardDeck()
	i := 0
	for deck.SizeDrawPile() > 0 {
		player := g.players[i%len(g.players)]
		player.pile = append(player.pile, deck.DrawCard())
		i++
	}
	g.turn = util.RandomInt(0, len(g.players)-1)
	pile_counts := make([]int, len(g.players))
	for i, p := range g.players {
		pile_counts[i] = len(p.pile)
	}
	game.Game_BroadcastUpdate(g, &game.UpdateMessage{Kind: "deal", Content: gin.H{
		"turn":        g.turn,
		"pile_counts": pile_counts,
	}})
	g.startTurnTimer()
}

func (g *GameEgyptianRatCrap) Valid() bool {
	if g.game == nil {
		return false
	}
	return true
}

func (g *GameEgyptianRatCrap) PlayerAction(action game.PlayerAction) {
	fmt.Println("player action:", action.Kind, action.Client_id, action.Action)
	switch action.Kind {
	case "timeout-flip":
		g.resolveTimeoutFlip()
		return
	case "resolve-challenge-award":
		g.resolveChallengeAward()
		return
	case "resolve-turn":
		g.resolveTurnAnnounce()
		return
	}
	player := g.game.Players[uint64(action.Client_id)]
	if player == nil {
		fmt.Fprintln(os.Stderr, "Invalid client id", action.Client_id)
		return
	}
	player_id := player.Player_id
	switch action.Kind {
	case "flip-card":
		if g.challenge_timer != nil || g.turn_announce_timer != nil {
			player.AddFailedUpdateShorthand("flip-card-failed", "Waiting for pile resolution")
			return
		}
		if player_id != g.turn {
			player.AddFailedUpdateShorthand("flip-card-failed",
				fmt.Sprintf("Not %d player's turn but %d player's turn", player_id, g.turn))
			return
		}
		if len(g.players[player_id].pile) == 0 {
			player.AddFailedUpdateShorthand("flip-card-failed", "No cards left to flip")
			return
		}
		g.executeFlipCard(player)
	case "slap":
		if g.players[player_id].slapped {
			player.AddFailedUpdateShorthand("slap-failed", "Already slapped this card")
			return
		}
		g.players[player_id].slapped = true
		g.executeSlap(player)
	default:
		fmt.Fprintln(os.Stderr, "Unknown game update type", action.Kind)
	}
}

func (g *GameEgyptianRatCrap) executeFlipCard(player *game.Player) {
	if g.turn_timer != nil {
		g.turn_timer.Stop()
		g.turn_timer = nil
	}
	flipping_player := g.players[g.turn]
	card := flipping_player.pile[len(flipping_player.pile)-1]
	flipping_player.pile = flipping_player.pile[:len(flipping_player.pile)-1]
	played_over_valid_slap := isValidSlap(g.central_pile)
	g.central_pile = append(g.central_pile, card)
	g.resetSlaps()

	update := &game.UpdateMessage{Kind: "flip-card", Content: gin.H{
		"card":                   card.ToFrontend(),
		"player_id":              player.Player_id,
		"played_over_valid_slap": played_over_valid_slap,
	}}

	busted := false
	if isChallengeCard(card) {
		g.challenge_active = true
		g.challenger_id = g.turn
		g.chances_left = chancesForCard(card)
		update.Content["challenger_id"] = g.challenger_id
		update.Content["chances_left"] = g.chances_left
		g.turn = g.nextActivePlayer(g.turn)
	} else if g.challenge_active {
		g.chances_left--
		if g.chances_left == 0 || len(flipping_player.pile) == 0 {
			busted = true
			update.Content["challenge_busted"] = true
			g.startChallengeGraceWindow()
		} else {
			update.Content["chances_left"] = g.chances_left
		}
	} else {
		g.turn = g.nextActivePlayer(g.turn)
	}

	if busted {
		update.Content["turn"] = -1
		game.Game_BroadcastUpdate(g, update)
		return
	}
	if winner_id := g.checkGameOver(); winner_id >= 0 {
		update.Content["turn"] = -1
		game.Game_BroadcastUpdate(g, update)
		g.game.EndGame(fmt.Sprintf("%s won the game", g.players[winner_id].player.GetNickname()))
		return
	}
	update.Content["turn"] = g.turn
	game.Game_BroadcastUpdate(g, update)
	g.startTurnTimer()
}

// Shuffles the central pile and places it on the bottom of the winner's pile
func (g *GameEgyptianRatCrap) awardCentralPileTo(winner *EgyptianRatCrapPlayer) int {
	util.Shuffle(g.central_pile)
	pile_size := len(g.central_pile)
	winner.pile = append(g.central_pile, winner.pile...)
	g.central_pile = []*game_utils.StandardCard{}
	return pile_size
}

func (g *GameEgyptianRatCrap) resolveSlapWin(player_id int) {
	if g.challenge_timer != nil {
		g.challenge_timer.Stop()
		g.challenge_timer = nil
	}
	g.challenge_active = false
	winner := g.players[player_id]
	pile_size := g.awardCentralPileTo(winner)
	game.Game_BroadcastUpdate(g, &game.UpdateMessage{Kind: "slap-result", Content: gin.H{
		"player_id": player_id,
		"valid":     true,
		"pile_size": pile_size,
	}})
	if winner_id := g.checkGameOver(); winner_id >= 0 {
		g.game.EndGame(fmt.Sprintf("%s won the game", g.players[winner_id].player.GetNickname()))
		return
	}
	g.startTurnAnnounce(player_id)
}

func (g *GameEgyptianRatCrap) executeSlap(player *game.Player) {
	if len(g.central_pile) == 0 {
		g.missedSlap(player.Player_id)
	} else if isValidSlap(g.central_pile) {
		g.resolveSlapWin(player.Player_id)
	} else {
		g.penalizeInvalidSlap(player.Player_id)
	}
}

func (g *GameEgyptianRatCrap) missedSlap(player_id int) {
	game.Game_BroadcastUpdate(g, &game.UpdateMessage{Kind: "slap-result", Content: gin.H{
		"player_id":  player_id,
		"valid":      false,
		"pile_empty": true,
	}})
}

func (g *GameEgyptianRatCrap) penalizeInvalidSlap(player_id int) {
	penalized := g.players[player_id]
	if len(penalized.pile) > 0 {
		card := penalized.pile[len(penalized.pile)-1]
		penalized.pile = penalized.pile[:len(penalized.pile)-1]
		g.central_pile = append(g.central_pile, card)
		g.resetSlaps()
	}
	game.Game_BroadcastUpdate(g, &game.UpdateMessage{Kind: "slap-result", Content: gin.H{
		"player_id": player_id,
		"valid":     false,
	}})
	if winner_id := g.checkGameOver(); winner_id >= 0 {
		g.game.EndGame(fmt.Sprintf("%s won the game", g.players[winner_id].player.GetNickname()))
	}
}

func (g *GameEgyptianRatCrap) startTurnAnnounce(winner_id int) {
	g.pending_turn_winner = winner_id
	timer := time.NewTimer(time.Second)
	g.turn_announce_timer = timer
	go func() {
		<-timer.C
		g.action_channel <- game.PlayerAction{Kind: "resolve-turn"}
	}()
}

func (g *GameEgyptianRatCrap) resolveTurnAnnounce() {
	if g.turn_announce_timer == nil {
		return
	}
	g.turn_announce_timer = nil
	g.turn = g.pending_turn_winner
	game.Game_BroadcastUpdate(g, &game.UpdateMessage{Kind: "turn-update", Content: gin.H{
		"turn": g.turn,
	}})
	g.startTurnTimer()
}

func (g *GameEgyptianRatCrap) startTurnTimer() {
	timer := time.NewTimer(2 * time.Second)
	g.turn_timer = timer
	go func() {
		<-timer.C
		g.action_channel <- game.PlayerAction{Kind: "timeout-flip"}
	}()
}

func (g *GameEgyptianRatCrap) resolveTimeoutFlip() {
	if g.turn_timer == nil {
		return
	}
	g.turn_timer = nil
	g.executeFlipCard(g.players[g.turn].player)
}

func (g *GameEgyptianRatCrap) startChallengeGraceWindow() {
	timer := time.NewTimer(time.Second)
	g.challenge_timer = timer
	go func() {
		<-timer.C
		g.action_channel <- game.PlayerAction{Kind: "resolve-challenge-award"}
	}()
}

func (g *GameEgyptianRatCrap) resolveChallengeAward() {
	if g.challenge_timer == nil {
		return
	}
	g.challenge_timer = nil
	winner := g.players[g.challenger_id]
	pile_size := g.awardCentralPileTo(winner)
	g.challenge_active = false
	game.Game_BroadcastUpdate(g, &game.UpdateMessage{Kind: "pile-awarded", Content: gin.H{
		"pile_awarded_to": g.challenger_id,
		"pile_size":       pile_size,
	}})
	if winner_id := g.checkGameOver(); winner_id >= 0 {
		g.game.EndGame(fmt.Sprintf("%s won the game", g.players[winner_id].player.GetNickname()))
		return
	}
	g.startTurnAnnounce(g.challenger_id)
}

func (g *GameEgyptianRatCrap) resetSlaps() {
	for _, p := range g.players {
		p.slapped = false
	}
}

// Advances from the given player, skipping anyone whose pile is empty
func (g *GameEgyptianRatCrap) nextActivePlayer(from int) int {
	next := from
	for i := 0; i < len(g.players); i++ {
		next = (next + 1) % len(g.players)
		if len(g.players[next].pile) > 0 {
			return next
		}
	}
	return next
}

// Returns the winning player_id, or -1 if the game isn't over
func (g *GameEgyptianRatCrap) checkGameOver() int {
	active := -1
	active_count := 0
	for i, p := range g.players {
		if len(p.pile) > 0 {
			active = i
			active_count++
		}
	}
	if active_count <= 1 {
		return active
	}
	return -1
}

func isValidSlap(central_pile []*game_utils.StandardCard) bool {
	return isDoubles(central_pile) || isSandwich(central_pile) || isSumToTen(central_pile) || isMarriage(central_pile)
}

func isDoubles(central_pile []*game_utils.StandardCard) bool {
	n := len(central_pile)
	if n < 2 {
		return false
	}
	return central_pile[n-1].GetNumber() == central_pile[n-2].GetNumber()
}

func isSandwich(central_pile []*game_utils.StandardCard) bool {
	n := len(central_pile)
	if n < 3 {
		return false
	}
	return central_pile[n-1].GetNumber() == central_pile[n-3].GetNumber()
}

func isSumToTen(central_pile []*game_utils.StandardCard) bool {
	n := len(central_pile)
	if n < 2 {
		return false
	}
	return central_pile[n-1].GetNumber()+central_pile[n-2].GetNumber() == 10
}

func isMarriage(central_pile []*game_utils.StandardCard) bool {
	n := len(central_pile)
	if n < 2 {
		return false
	}
	top, second := central_pile[n-1].GetNumber(), central_pile[n-2].GetNumber()
	return (top == 12 && second == 13) || (top == 13 && second == 12)
}

func isChallengeCard(card *game_utils.StandardCard) bool {
	return card.GetNumber() >= 11 // jack, queen, king, ace
}

func chancesForCard(card *game_utils.StandardCard) uint8 {
	switch card.GetNumber() {
	case 11: // jack
		return 1
	case 12: // queen
		return 2
	case 13: // king
		return 3
	case 14: // ace
		return 4
	default:
		return 0
	}
}

func (g *GameEgyptianRatCrap) PlayerDisconnected(client_id uint64) {
}

func (g *GameEgyptianRatCrap) PlayerReconnected(client_id uint64) {
}

func (g *GameEgyptianRatCrap) ToFrontend(client_id uint64, is_viewer bool) gin.H {
	game := gin.H{
		"turn":             g.turn,
		"challenge_active": g.challenge_active,
		"challenger_id":    g.challenger_id,
		"chances_left":     g.chances_left,
		"pile_pending":     g.challenge_timer != nil || g.turn_announce_timer != nil,
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
	central_pile := []gin.H{}
	for _, card := range g.central_pile {
		if card != nil {
			central_pile = append(central_pile, card.ToFrontend())
		}
	}
	game["central_pile"] = central_pile
	return game
}
