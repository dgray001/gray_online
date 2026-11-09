package neural

import "github.com/dgray001/gray_online/game/games/fiddlesticks"

func progressFeatures(input []float64, s fiddlesticks.LearningState) {
	i := input[726:]
	p := s.Progress
	denominator := float64(max(1, p.Total-1))
	maximum := max(1, p.MaximumPoints)
	i[0], i[1], i[2], i[3] = float64(s.MinRound)/26, float64(s.MaxRound)/26, float64(p.Index)/denominator, float64(p.Remaining)/denominator
	if s.Increasing {
		i[4] = 1
	}
	i[5] = float64(s.Iteration) / float64(max(1, s.Iterations))
	i[6] = p.FuturePoints / maximum
	leader, opponent, higher := uint32(0), uint32(0), 0
	for seat, score := range s.Scores {
		leader = max(leader, score)
		if seat != s.Seat {
			opponent = max(opponent, score)
		}
		if score > s.Scores[s.Seat] {
			higher++
		}
	}
	if len(s.Scores) > s.Seat {
		i[7], i[8] = (float64(leader)-float64(s.Scores[s.Seat]))/maximum, (float64(s.Scores[s.Seat])-float64(opponent))/maximum
	}
	i[9] = float64(higher) / float64(max(1, len(s.Bets)-1))
	remaining := len(s.Hand)
	for _, card := range s.Hand {
		for _, seen := range s.Seen {
			if card.Hash() == seen.Hash() {
				remaining--
				break
			}
		}
	}
	i[10] = float64(remaining) / 26
	if len(s.Bets) > s.Seat {
		i[11] = (float64(s.Bets[s.Seat]) - float64(s.Tricks[s.Seat])) / 26
	}
	if s.Trump != nil {
		i[12] = float64(s.Trump.GetNumber()) / 14
	}
	i[13] = float64(len(s.Trick)) / float64(max(1, len(s.Bets)))
}
