package goldset

import (
	"crypto/sha256"
	"fiddlesticks.live/neural"
	"fmt"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
	"time"
)

func capture(q GenerateRequest, source Source, p *fiddlesticks.FiddlesticksPlayer, f *fiddlesticks.GameFiddlesticks, model *fiddlesticks.FiddlesticksAiModelTheory4, pools map[string]*reservoir) *Item {
	s := f.LearningState(p)
	kind := "play-card"
	if f.IsBetting() {
		kind = "bet"
	}
	if s.Round < q.MinHand || s.Round > q.MaxHand {
		return nil
	}
	slot := pools[kind].next()
	if slot < 0 {
		return nil
	}
	explanation := model.ExplainDecision(p, f)
	legal := []int{}
	if kind == "bet" {
		for i := 0; i <= int(s.Round); i++ {
			legal = append(legal, i)
		}
	} else {
		for _, card := range explanation.Cards {
			legal = append(legal, 27+int(card.Suit-1)*13+int(card.Number-2))
		}
	}
	key := fmt.Sprintf("%d/%d/%d/%d/%d/%d", source.Seed, source.Players, source.RoundPoints, source.TrickPoints, s.Seat, source.Decision)
	hash := sha256.Sum256([]byte(key))
	return &Item{slot: slot, SchemaVersion: neural.FeatureVersion, ID: fmt.Sprintf("%x", hash[:16]), CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), Source: source, State: snapshot(s, kind), Features: neural.Features(s, kind == "bet"), Legal: legal, Analysis: explanation.ToFrontend(), Model4Parameters: model.ModelParameters()}
}
