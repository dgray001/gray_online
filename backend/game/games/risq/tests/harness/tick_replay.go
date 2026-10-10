package harness

import (
	"encoding/json"
	"testing"
)

func DecodeSnapshot(t *testing.T, value any) State {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	return state
}

type ReplayFrame struct {
	Tick      int          `json:"tick"`
	Terminal  bool         `json:"terminal"`
	Units     []Unit       `json:"units"`
	Buildings []Building   `json:"buildings"`
	Effects   []TickEffect `json:"effects"`
}

type TurnReplay struct {
	SchemaVersion  int           `json:"schema_version"`
	TurnNumber     int           `json:"turn_number"`
	TickCount      int           `json:"tick_count"`
	Baseline       ReplayFrame   `json:"baseline"`
	Frames         []ReplayFrame `json:"frames"`
	Units          []Unit        `json:"units"`
	BoundaryEvents []ReplayFrame `json:"boundary_events"`
}

func RequireReplay(t *testing.T, state State) *TurnReplay {
	t.Helper()
	if state.LastTurnReplay == nil {
		t.Fatal("missing last_turn_replay")
	}
	return state.LastTurnReplay
}

func (r *TurnReplay) Unit(id uint64) *Unit {
	for i := range r.Units {
		if r.Units[i].InternalID == id {
			return &r.Units[i]
		}
	}
	return nil
}

func (r *TurnReplay) UnitAt(id uint64, tick int) *Unit {
	var found *Unit
	frames := append([]ReplayFrame{r.Baseline}, r.Frames...)
	for _, frame := range frames {
		if frame.Tick > tick {
			continue
		}
		for i := range frame.Units {
			if frame.Units[i].InternalID == id {
				found = &frame.Units[i]
			}
		}
	}
	return found
}

func (r *TurnReplay) Frame(t *testing.T, tick int) ReplayFrame {
	t.Helper()
	for _, frame := range r.Frames {
		if frame.Tick == tick {
			return frame
		}
	}
	t.Fatalf("missing replay frame %d", tick)
	return ReplayFrame{}
}
