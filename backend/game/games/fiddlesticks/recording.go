package fiddlesticks

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

type gameRecording struct {
	id     string
	events []json.RawMessage
}

func (f *GameFiddlesticks) recordAction(player *game.Player, action game.PlayerAction) {
	p := f.players[player.Player_id]
	if f.recording != nil && player.IsHumanPlayer() && p.ai_choice == nil {
		p.ai_last_applied_id = player.ReplayUpdates(p.ai_last_applied_id, func(update *game.UpdateMessage) {
			p.ai_model.ApplyUpdate(p, f, update)
		})
		if f.betting {
			p.ai_choice = &aiChoice{model_id: p.ai_model_id, bet: GetAiBid(p, f)}
		} else {
			p.ai_choice = &aiChoice{model_id: p.ai_model_id, card: p.cards[GetAiPlayCard(p, f)]}
		}
	}
	source := "human"
	if !player.IsHumanPlayer() {
		source = "bot"
	} else if isTimeoutAction(action) {
		source = "timeout"
	}
	f.recordEvent("action", gin.H{"seat": player.Player_id, "kind": action.Kind, "action": action.Action, "source": source})
}

func (f *GameFiddlesticks) recordDeal() {
	if f.recording == nil {
		return
	}
	hands := make([][]gin.H, len(f.players))
	for seat, player := range f.players {
		for _, card := range player.cards {
			hands[seat] = append(hands[seat], card.ToFrontend())
		}
	}
	_, scores := f.GetGameResults()
	f.recordEvent("deal", gin.H{"round": f.round, "increasing": f.rounds_increasing, "dealer": f.dealer, "trump": f.trump.ToFrontend(), "hands": hands, "scores": scores})
	f.checkpointRecording()
}

func modelParameters(model FiddlesticksAiModel) gin.H {
	parameters := gin.H{"type": fmt.Sprintf("%T", model)}
	if m, ok := model.(*FiddlesticksAiModelTheory4); ok {
		parameters["values"] = m.recordingParameters()
	}
	return parameters
}

func (f *GameFiddlesticks) startRecording() {
	if os.Getenv("GAE_SERVICE") == "" || os.Getenv("FIDDLESTICKS_RECORDING_BUCKET") == "" {
		return
	}
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		util.Errorf(f.game.Err, "Fiddlesticks recording ID: %v", err)
		return
	}
	f.recording = &gameRecording{id: hex.EncodeToString(id)}
	f.recordEvent("start", gin.H{"schema_version": 1, "deployment": os.Getenv("GAE_VERSION"), "settings": f.game.GameSpecificSettings, "game_id": f.game.Game_id})
	f.recordEvent("settings", gin.H{"min_round": f.min_round, "max_round": f.max_round, "round_points": f.round_points, "trick_points": f.trick_points, "iterations": f.iterations, "turn_duration_ms": f.turn_duration.Milliseconds()})
	for _, p := range f.players {
		f.recordEvent("player", gin.H{"seat": p.player.Player_id, "nickname": p.player.GetNickname(), "client_id": p.player.GetClientId(), "human": p.player.IsHumanPlayer(), "identity": p.player.Identity, "model_id": p.ai_model_id, "parameters": modelParameters(p.ai_model)})
	}
}

func (f *GameFiddlesticks) recordEvent(kind string, data gin.H) {
	if f.recording == nil {
		return
	}
	event, err := json.Marshal(gin.H{"sequence": len(f.recording.events) + 1, "time": time.Now().UTC(), "kind": kind, "data": data})
	if err != nil {
		util.Errorf(f.game.Err, "Fiddlesticks recording event: %v", err)
		return
	}
	f.recording.events = append(f.recording.events, event)
}

func (f *GameFiddlesticks) checkpointRecording() {
	if f.recording == nil {
		return
	}
	data, err := json.Marshal(gin.H{"recording_id": f.recording.id, "events": f.recording.events})
	if err != nil {
		util.Errorf(f.game.Err, "Fiddlesticks checkpoint: %v", err)
		return
	}
	name := fmt.Sprintf("fiddlesticks/%s/%06d.json", f.recording.id, len(f.recording.events))
	go uploadRecording(name, data)
}

func uploadRecording(name string, data []byte) {
	for attempt := 0; attempt < 3; attempt++ {
		if err := writeRecording(name, data); err == nil {
			return
		} else {
			util.Errorf(util.DefaultErr, "Fiddlesticks upload %s attempt %d: %v", name, attempt+1, err)
		}
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}
}

func writeRecording(name string, data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	writer := client.Bucket(os.Getenv("FIDDLESTICKS_RECORDING_BUCKET")).Object(name).If(storage.Conditions{DoesNotExist: true}).NewWriter(ctx)
	writer.ContentType = "application/json"
	writer.ChunkSize = 0
	writer.CacheControl = "no-store"
	_, err = writer.Write(data)
	return errors.Join(err, writer.Close())
}
