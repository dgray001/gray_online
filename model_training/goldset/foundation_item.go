package goldset

import (
	"fmt"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
	"math/rand"
)

func foundationItem(spec FoundationCase, seed int64, index int) (Item, error) {
	f, err := prepareFoundation(spec, seed)
	if err != nil {
		return Item{}, err
	}
	p := f.GetFiddlesticksPlayer(spec.Seat)
	model := p.AiModel().(*fiddlesticks.FiddlesticksAiModelTheory4)
	source := Source{PresetID: fmt.Sprintf("foundation-002/%02d", index), Seed: seed, ActionSeed: seed + 7919, Decision: len(spec.Plays) + 1, Players: spec.Players, RoundPoints: spec.RoundPoints, TrickPoints: spec.TrickPoints}
	pool := map[string]*reservoir{spec.Kind: {Limit: 1, Random: rand.New(rand.NewSource(seed))}}
	q := GenerateRequest{MinHand: uint8(len(spec.Hand)), MaxHand: uint8(len(spec.Hand))}
	item := capture(q, source, p, f, model, pool)
	if item == nil {
		return Item{}, fmt.Errorf("capture failed")
	}
	if spec.Kind == "bet" {
		item.Model4Action = int(fiddlesticks.GetAiBid(p, f))
	} else {
		chosen := fiddlesticks.GetAiPlayCard(p, f)
		item.Model4Action = item.State.Hand[chosen].Action
	}
	return *item, nil
}
