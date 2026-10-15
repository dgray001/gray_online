package bridge

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/game/games/risq/internal/aibridge"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/config"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

var mapFile string

func TestMain(m *testing.M) {
	cleanup := setupConfigDir()
	defer cleanup()
	os.Exit(m.Run())
}

func setupConfigDir() func() {
	dir, err := os.MkdirTemp("", "bridge-*")
	if err != nil {
		panic(err)
	}
	mapDir := filepath.Join(dir, "maps", "custom")
	_ = os.MkdirAll(mapDir, 0o755)
	mapFile = filepath.Join(mapDir, "bridge-test.json")
	copyFiles(dir)
	_ = defs.LoadConfig(dir)
	return func() { _ = defs.LoadConfig(testconfig.Dir()); _ = os.RemoveAll(dir) }
}

func copyFiles(dir string) {
	files := []string{"ai/default.json", "bonuses.json", "buildings.json", "region_names.json", "resources.json", "techs.json", "terrains.json", "units.json"}
	for _, f := range files {
		data, _ := os.ReadFile(filepath.Join(testconfig.Dir(), f))
		dst := filepath.Join(dir, f)
		_ = os.MkdirAll(filepath.Dir(dst), 0o755)
		_ = os.WriteFile(dst, data, 0o644)
	}
}

func bridgeGame(t *testing.T, spaces string, configure ...func()) (*harness.Game, *game.Player) {
	t.Helper()
	document := `{"board_size":4,"players":2,"spaces":[` + spaces + `]}`
	if err := os.WriteFile(mapFile, []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	if len(configure) > 0 {
		saved := defs.UnitConfigs[11]
		t.Cleanup(func() { defs.UnitConfigs[11] = saved })
		for _, apply := range configure {
			apply()
		}
	}
	g := harness.NewGame(t, "custom:bridge-test", 1, 2)
	player := game.CreateAiPlayer("probe", g.Base)
	player.Player_id = 0
	return g, player
}

func receive[T any](t *testing.T, channel <-chan T) T {
	t.Helper()
	select {
	case value := <-channel:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for runner")
	}
	var zero T
	return zero
}

func parsedModel(t *testing.T, actions string) ai.Model {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal([]byte(`{"rules":[{"when":{"always":{}},"then":[`+actions+`]}]}`), &raw); err != nil {
		t.Fatal(err)
	}
	model := ai.ParseModel(raw, rand.New(rand.NewSource(1)))
	if _, ok := model.(*ai.RulesModel); !ok {
		t.Fatal("invalid fixture model")
	}
	return model
}

type observation struct {
	Units            []ai.UnitView
	VisibleBuildings []ai.BuildingView
	KnownBuildings   []ai.BuildingView
	Resources        []ai.ResourceView
	Spaces           []ai.SpaceInfo
}

type recordingModel struct {
	ai.Model
	Seen chan observation
}

func (m *recordingModel) DecideOrders(view ai.View) ai.Decision {
	m.Seen <- observation{Units: view.VisibleEnemyUnits(), VisibleBuildings: view.VisibleEnemyBuildings(), KnownBuildings: view.KnownEnemyBuildings(), Resources: view.KnownResources(ai.ResourceFood), Spaces: view.AllSpaces()}
	return m.Model.DecideOrders(view)
}

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

func assertEmptySubmission(t *testing.T, action game.PlayerAction) {
	t.Helper()
	if action.Kind != "submit-orders" || action.Ai_id != 1 || action.Client_id != 0 || len(action.Action["orders"].([]defs.OrderFromFrontend)) != 0 {
		t.Fatalf("fallback=%+v", action)
	}
}

func TestRunnerFallbackOncePerTurn(t *testing.T) {
	g, player := bridgeGame(t, ownScout+`,{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}`)
	player.AiFailedUpdates = make(chan *game.UpdateMessage)
	w := runWorker(t, player, ai.NoopModel{}, make(chan game.PlayerAction, 8))
	update := &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": g.Risq.ToFrontend(uint64(g.Human(0)+1), false)}}
	failure := &game.UpdateMessage{Kind: "submit-orders-failed", Content: gin.H{"message": "test rejection"}}
	player.AiUpdates <- update
	assertEmptySubmission(t, receive(t, w.Actions))
	player.AiFailedUpdates <- failure
	assertEmptySubmission(t, receive(t, w.Actions))
	player.AiFailedUpdates <- failure
	player.AiFailedUpdates <- failure
	player.AiUpdates <- update
	assertEmptySubmission(t, receive(t, w.Actions))
	player.AiFailedUpdates <- failure
	assertEmptySubmission(t, receive(t, w.Actions))
	w.close(t)
	if len(w.Actions) != 0 {
		t.Fatalf("extra fallback submissions: %d", len(w.Actions))
	}
}

func TestRunnerIgnoresOtherUpdatesAndSubmitsMalformedSnapshot(t *testing.T) {
	_, player := bridgeGame(t, ownScout+`,{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}`)
	player.AiUpdates = make(chan *game.UpdateMessage)
	seen := &recordingModel{Model: ai.NoopModel{}, Seen: make(chan observation, 2)}
	w := runWorker(t, player, seen, make(chan game.PlayerAction, 8))
	player.AiUpdates <- &game.UpdateMessage{Kind: "submitted-orders"}
	player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": gin.H{"turn_number": "bad"}}}
	assertEmptySubmission(t, receive(t, w.Actions))
	player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": gin.H{}}}
	assertEmptySubmission(t, receive(t, w.Actions))
	w.close(t)
	if len(seen.Seen) != 0 || len(w.Actions) != 0 {
		t.Fatal("invalid or unrelated update reached the model")
	}
}

func TestRunnerStopsWaitingAndSending(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		t.Run(map[bool]string{false: "waiting", true: "sending"}[blocked], func(t *testing.T) {
			g, player := bridgeGame(t, ownScout+`,{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}`)
			seen := &recordingModel{Model: ai.NoopModel{}, Seen: make(chan observation, 1)}
			w := runWorker(t, player, seen, make(chan game.PlayerAction))
			if blocked {
				payload := g.Risq.ToFrontend(uint64(g.Human(0)+1), false)
				player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": payload}}
				receive(t, seen.Seen)
			}
			w.close(t)
			select {
			case action := <-w.Actions:
				t.Fatalf("post-stop action=%+v", action)
			default:
			}
		})
	}
}
