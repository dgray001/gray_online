package tournament

import "math"

// Multi-player Elo calculator using the standard Plackett-Luce model / pairwise decomposition.
// Can be used for AIs in model_training and reused directly for human players in website backend.

type PlayerRating struct {
	ID       string  `json:"id"`
	Rating   float64 `json:"rating"`
	Games    int     `json:"games"`
	Wins     int     `json:"wins"`
	RankSums float64 `json:"rank_sums"`
}

func NewPlayerRating(id string, initialRating float64) *PlayerRating {
	if initialRating <= 0 {
		initialRating = 1500.0
	}
	return &PlayerRating{
		ID:     id,
		Rating: initialRating,
	}
}

// UpdateRatings updates player ratings in-place following a game finish.
// placements is sorted from 1st place to last place. Ties share equal placement ranks.
func UpdateRatings(ratings []*PlayerRating, scores []float64, kFactor float64) {
	n := len(ratings)
	if n < 2 || len(scores) != n {
		return
	}

	deltas := make([]float64, n)
	scale := 1.0 / float64(n-1)

	for i := 0; i < n; i++ {
		ratings[i].Games++
		if scores[i] == maxSlice(scores) {
			ratings[i].Wins++
		}
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}

			// Expected score of player i against player j
			ea := 1.0 / (1.0 + math.Pow(10.0, (ratings[j].Rating-ratings[i].Rating)/400.0))

			// Actual score: 1.0 for win, 0.5 for tie, 0.0 for loss
			actual := 0.5
			if scores[i] > scores[j] {
				actual = 1.0
			} else if scores[i] < scores[j] {
				actual = 0.0
			}

			deltas[i] += kFactor * (actual - ea) * scale
		}
	}

	for i := 0; i < n; i++ {
		ratings[i].Rating += deltas[i]
	}
}

func maxSlice(s []float64) float64 {
	m := s[0]
	for _, v := range s[1:] {
		if v > m {
			m = v
		}
	}
	return m
}
