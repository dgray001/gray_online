package aisim

import (
	"testing"
	"time"
)

func (g *Game) Stop(t *testing.T) {
	t.Helper()
	done := make(chan struct{})
	go func() { g.Risq.StopAi(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("StopAi did not finish")
	}
}
