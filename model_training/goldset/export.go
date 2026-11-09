package goldset

import "fiddlesticks.live/neural"

type GoldExample struct {
	Acceptable []int            `json:"acceptable_actions"`
	Reference  *Model4Reference `json:"model4_reference"`
	ID         string           `json:"id"`
	Source     Source           `json:"source"`
	State      State            `json:"state"`
	Features   []float64        `json:"features"`
	Legal      []int            `json:"legal"`
	Target     int              `json:"target_action"`
	Confidence float64          `json:"confidence"`
	Notes      string           `json:"notes"`
	SavedAt    string           `json:"saved_at"`
}
type Goldset struct {
	LabelVersion int           `json:"label_schema_version"`
	Version      int           `json:"schema_version"`
	Inputs       int           `json:"input_count"`
	Objective    string        `json:"objective"`
	Examples     []GoldExample `json:"examples"`
}

func (s *Store) Export() Goldset {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := Goldset{Version: neural.FeatureVersion, LabelVersion: 2, Inputs: neural.Inputs, Objective: "win", Examples: []GoldExample{}}
	for _, id := range s.order {
		label, exists := s.latest[id]
		if !exists {
			continue
		}
		item := s.items[id]
		result.Examples = append(result.Examples, GoldExample{ID: id, Source: item.Source, State: item.State, Features: item.Features, Legal: item.Legal, Target: label.Action, Acceptable: label.Acceptable, Confidence: label.Confidence, Notes: label.Notes, SavedAt: label.SavedAt, Reference: label.Reference})
	}
	return result
}
