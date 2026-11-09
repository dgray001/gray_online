package fiddlesticks

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/gin-gonic/gin"
)

func TestRecordingSnapshotsMutableWeights(t *testing.T) {
	f := &GameFiddlesticks{recording: &gameRecording{id: "test"}}
	weights := []float64{0.25, 0.75}
	f.recordEvent("prediction", gin.H{"weights": weights})
	weights[0] = 99
	var event struct{ Data struct{ Weights []float64 } }
	if err := json.Unmarshal(f.recording.events[0], &event); err != nil {
		t.Fatal(err)
	}
	if event.Data.Weights[0] != 0.25 {
		t.Fatal("recording changed after model weights mutated")
	}
}

func TestRecordingStorageUpload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil || !strings.Contains(string(body), `{"recording_id":"test"}`) {
			t.Error("missing recording payload", err)
		}
		if r.URL.Query().Get("name") != "fiddlesticks/test/000001.json" || r.URL.Query().Get("ifGenerationMatch") != "0" {
			t.Error("missing immutable object condition", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{}`)
	}))
	defer server.Close()
	t.Setenv("STORAGE_EMULATOR_HOST", server.URL)
	t.Setenv("FIDDLESTICKS_RECORDING_BUCKET", "test-bucket")
	if err := writeRecording("fiddlesticks/test/000001.json", []byte(`{"recording_id":"test"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestRecordingPredictsBeforeFastHumanAction(t *testing.T) {
	base := game.CreateBaseGame(1, game.GameType_FIDDLESTICKS, nil, nil, nil)
	game.CreatePlayer(1, "human", base)
	player := base.Players[1]
	player.Player_id = 0
	p := &FiddlesticksPlayer{player: player, ai_model: &FiddlesticksAiModelRandom{}}
	f := &GameFiddlesticks{game: base, players: []*FiddlesticksPlayer{p}, betting: true, round: 1, recording: &gameRecording{id: "test"}}
	f.recordAction(player, game.PlayerAction{Kind: "bet", Action: gin.H{"amount": float64(0)}})
	if len(f.recording.events) != 2 || p.ai_choice == nil {
		t.Fatal("missing prediction before action")
	}
	var event struct{ Kind string }
	if err := json.Unmarshal(f.recording.events[0], &event); err != nil {
		t.Fatal(err)
	}
	if event.Kind != "prediction" {
		t.Fatal(event.Kind)
	}
}
