package harness

import (
	"math"
	"testing"
)

type TickEffect struct {
	Kind   string     `json:"kind"`
	Actor  TickTarget `json:"actor"`
	Target TickTarget `json:"target"`
	Amount float64    `json:"amount"`
}

func RequireTickAction(t *testing.T, actions []TickAction, kind string) TickAction {
	t.Helper()
	for _, action := range actions {
		if action.Intent.Kind == kind {
			return action
		}
	}
	t.Fatalf("missing %s tick action in %+v", kind, actions)
	return TickAction{}
}

func RequireTickBlocked(t *testing.T, actions []TickAction, reason string) TickAction {
	t.Helper()
	for _, action := range actions {
		if action.Execute.Reason == reason && action.Execute.Outcome == "blocked" && action.Execute.StaminaSpent == 0 {
			return action
		}
	}
	t.Fatalf("missing zero-spend blocked outcome %s in %+v", reason, actions)
	return TickAction{}
}

func AssertTickSpend(t *testing.T, actions []TickAction, want int) {
	t.Helper()
	spent := 0
	for _, action := range actions {
		if action.Tick < 1 || action.Execute.StaminaSpent < 0 {
			t.Errorf("invalid tick/spend: %+v", action)
		}
		spent += action.Execute.StaminaSpent
	}
	if spent != want {
		t.Errorf("history spent %d stamina, want %d", spent, want)
	}
}

func AssertTickEffect(t *testing.T, actions []TickAction, kind string, target TickTarget, want float64) {
	t.Helper()
	amount, count := 0.0, 0
	for _, action := range actions {
		for _, effect := range action.Execute.Effects {
			if effect.Kind == kind && effect.Target == target {
				amount += effect.Amount
				count++
			}
		}
	}
	if count == 0 || math.Abs(amount-want) > 0.0001 {
		t.Errorf("history %s on %+v: %d effects totaling %v, want %v", kind, target, count, amount, want)
	}
}

type TickLocation struct {
	Space Coord `json:"space"`
	Zone  Coord `json:"zone"`
}

type TickResolution struct {
	Kind             string `json:"kind"`
	Reason           string `json:"reason"`
	StaminaAllocated int    `json:"stamina_allocated"`
	SunkCost         int    `json:"sunk_cost"`
}

type TickTarget struct {
	Kind       string `json:"kind"`
	InternalID uint64 `json:"internal_id"`
}

type TickIntent struct {
	Location         TickLocation   `json:"location"`
	Destination      TickLocation   `json:"destination"`
	TargetLocation   TickLocation   `json:"target_location"`
	AvailableStamina int            `json:"available_stamina"`
	MinCost          int            `json:"min_cost"`
	MaxCost          int            `json:"max_cost"`
	Resolution       TickResolution `json:"resolution"`

	Kind   string     `json:"kind"`
	Target TickTarget `json:"target"`
}

type TickExecution struct {
	TargetLocation TickLocation `json:"target_location"`
	Effects        []TickEffect `json:"effects"`
	Gathered       float64      `json:"gathered"`
	Progress       int          `json:"progress"`

	Outcome      string     `json:"outcome"`
	Reason       string     `json:"reason"`
	StaminaSpent int        `json:"stamina_spent"`
	Target       TickTarget `json:"target"`
	Healing      float64    `json:"healing"`
	WoodSpent    float64    `json:"wood_spent"`
}

type TickAction struct {
	Sequence int `json:"sequence"`

	Tick    int           `json:"tick"`
	Intent  TickIntent    `json:"intent"`
	Execute TickExecution `json:"execute"`
	Order   struct {
		OrderType uint8  `json:"order_type"`
		TargetID  int64  `json:"target_id"`
		Source    string `json:"source"`
	} `json:"order"`
}
