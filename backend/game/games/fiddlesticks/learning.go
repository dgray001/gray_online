package fiddlesticks

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"math/rand"
)

type LearningState struct {
	History       []PublicAction
	RoundHistory  []PublicAction
	Voids         [][4]bool
	Profiles      []PublicProfile
	PriorProfiles []PublicProfile
	Progress      GameProgress
	MinRound      uint8
	MaxRound      uint8
	Increasing    bool
	TrickLeader   int
	Iteration     uint32
	Iterations    uint32
	Hand          []*game_utils.StandardCard
	Seen          []*game_utils.StandardCard
	Trick         []*game_utils.StandardCard
	Trump         *game_utils.StandardCard
	Round         uint8
	Seat          int
	Dealer        int
	Bets          []uint8
	Tricks        []uint8
	Scores        []uint32
	HasBet        []bool
	RoundPoints   uint16
	TrickPoints   uint16
}

func (p *FiddlesticksPlayer) AiModel() FiddlesticksAiModel         { return p.ai_model }
func (p *FiddlesticksPlayer) SetAiModel(model FiddlesticksAiModel) { p.ai_model = model }

func (f *GameFiddlesticks) SetDealSeed(seed int64) { f.deal_random = rand.New(rand.NewSource(seed)) }

func (f *GameFiddlesticks) SetActionSeed(seed int64) {
	f.action_random = rand.New(rand.NewSource(seed))
}

func (f *GameFiddlesticks) randomFloat() float64 {
	if f.action_random != nil {
		return f.action_random.Float64()
	}
	return rand.Float64()
}

func (f *GameFiddlesticks) LearningState(p *FiddlesticksPlayer) LearningState {
	state := LearningState{Hand: p.cards, Trick: f.trick, Trump: f.trump, Round: f.round, Seat: p.player.Player_id, Dealer: f.dealer}
	state.RoundPoints, state.TrickPoints = f.round_points, f.trick_points
	state.History, state.RoundHistory = f.public_history, f.round_history
	state.Voids, state.Profiles, state.PriorProfiles = append([][4]bool(nil), f.known_voids...), append([]PublicProfile(nil), f.public_profiles...), append([]PublicProfile(nil), f.prior_profiles...)
	state.Progress, state.MinRound, state.MaxRound, state.Increasing = f.progress(), f.min_round, f.max_round, f.rounds_increasing
	state.TrickLeader, state.Iteration, state.Iterations = f.trick_leader, f.iteration, f.iterations
	for _, player := range f.players {
		state.Bets = append(state.Bets, player.bet)
		state.Tricks = append(state.Tricks, player.tricks)
		state.Scores = append(state.Scores, player.score)
		state.HasBet = append(state.HasBet, player.has_bet)
		for _, index := range player.cards_played {
			state.Seen = append(state.Seen, player.cards[index])
		}
	}
	return state
}
