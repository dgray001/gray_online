package bridge

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/game/games/risq/internal/aibridge"
	"testing"
	"time"
)

type worker struct {
	Stop    chan struct{}
	Done    chan struct{}
	Actions chan game.PlayerAction
}

func runWorker(t *testing.T, player *game.Player, model ai.Model, actions chan game.PlayerAction) *worker {
	t.Helper()
	w := &worker{Stop: make(chan struct{}), Done: make(chan struct{}), Actions: actions}
	runner := &aibridge.Runner{Player: player, Model: model, Stop: w.Stop, Actions: actions}
	go func() { runner.Run(); close(w.Done) }()
	t.Cleanup(func() { w.close(t) })
	return w
}

func (w *worker) close(t *testing.T) {
	t.Helper()
	select {
	case <-w.Stop:
	default:
		close(w.Stop)
	}
	select {
	case <-w.Done:
	case <-time.After(5 * time.Second):
		t.Fatal("runner did not exit")
	}
}
