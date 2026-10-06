package main

import "github.com/dgray001/gray_online/game/games/risq/internal/simoutcome"

type Outcome = simoutcome.Outcome

var (
	outcomeLetters = simoutcome.Letters
	outcomeOrder   = simoutcome.Order
	outcomes       = simoutcome.Outcomes
	finalOutcomes  = simoutcome.FinalOutcomes
)

const (
	OutcomeDefeat   = simoutcome.OutcomeDefeat
	OutcomeCrush    = simoutcome.OutcomeCrush
	OutcomeWinning  = simoutcome.OutcomeWinning
	OutcomeAhead    = simoutcome.OutcomeAhead
	OutcomeBehind   = simoutcome.OutcomeBehind
	OutcomeLosing   = simoutcome.OutcomeLosing
	OutcomeCrushed  = simoutcome.OutcomeCrushed
	OutcomeDefeated = simoutcome.OutcomeDefeated
)
