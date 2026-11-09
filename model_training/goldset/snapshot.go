package goldset

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
)

func cardView(card *game_utils.StandardCard, index int) Card {
	return Card{Suit: card.GetSuit(), Number: card.GetNumber(), Index: index, Action: 27 + int(card.GetSuit()-1)*13 + int(card.GetNumber()-2)}
}

func snapshot(s fiddlesticks.LearningState, kind string) State {
	p := s.Progress
	state := State{Kind: kind, Objective: "win", Seat: s.Seat, Dealer: s.Dealer, TrickLeader: s.TrickLeader, HandSize: s.Round, MaxHand: s.MaxRound, Increasing: s.Increasing, Progress: Progress{p.Index, p.Total, p.Remaining, p.FuturePoints}, RoundPoints: s.RoundPoints, TrickPoints: s.TrickPoints, Trump: cardView(s.Trump, -1)}
	seen := map[uint16]bool{}
	for _, card := range s.Seen {
		seen[card.Hash()] = true
	}
	for index, card := range s.Hand {
		c := cardView(card, index)
		c.Played = seen[card.Hash()]
		state.Hand = append(state.Hand, c)
	}
	for _, card := range s.Trick {
		state.Trick = append(state.Trick, cardView(card, -1))
	}
	for seat := range s.Bets {
		player := PublicPlayer{Seat: seat, Score: s.Scores[seat], Bid: s.Bets[seat], HasBid: s.HasBet[seat], Tricks: s.Tricks[seat]}
		if seat < len(s.Voids) {
			player.Voids = s.Voids[seat]
		}
		if seat < len(s.Profiles) {
			player.Profile = profileView(s.Profiles[seat])
		}
		if seat < len(s.PriorProfiles) {
			player.PriorProfile = profileView(s.PriorProfiles[seat])
		}
		state.Players = append(state.Players, player)
	}
	for _, e := range s.History {
		state.History = append(state.History, Event{e.Seat, e.RoundIndex, e.HandSize, e.Trick, e.Order, e.Kind, e.Suit, e.Number, e.Bid})
	}
	return state
}
