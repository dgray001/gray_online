package orders

import (
	"fmt"
	"slices"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func queuedUpdates(g *harness.Game) string {
	counts := []int{len(g.Base.ViewerUpdates)}
	var clients []uint64
	for id := range g.Base.Players {
		clients = append(clients, id)
	}
	slices.Sort(clients)
	for _, id := range clients {
		counts = append(counts, len(g.Base.Players[id].Updates))
	}
	return fmt.Sprint(counts)
}
