package harness

import "encoding/json"

func (g *Game) playerStates(i int) []PlayerState {
	g.T.Helper()
	data, err := json.Marshal(g.Risq.ToFrontend(g.clients[i], false)["players"])
	if err != nil {
		g.T.Fatal(err)
	}
	var players []PlayerState
	if err := json.Unmarshal(data, &players); err != nil {
		g.T.Fatal(err)
	}
	return players
}
