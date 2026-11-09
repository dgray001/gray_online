package goldset

import "fmt"

func (s *Store) normalizeAcceptable(corrections map[string][]int) error {
	latest := map[string]int{}
	for i, label := range s.labels {
		latest[label.ItemID] = i
	}
	for id := range corrections {
		if _, exists := latest[id]; !exists {
			return fmt.Errorf("no label found for equivalence correction %s", id)
		}
	}
	for i := range s.labels {
		label := &s.labels[i]
		item, exists := s.items[label.ItemID]
		if !exists {
			return fmt.Errorf("item missing for %s", label.ItemID)
		}
		actions := label.Acceptable
		if proposed, exists := corrections[label.ItemID]; exists && latest[label.ItemID] == i {
			actions = proposed
		}
		normalized, err := acceptableActions(item, label.Action, actions)
		if err != nil {
			return fmt.Errorf("%s: %w", label.ItemID, err)
		}
		label.Acceptable = normalized
	}
	return nil
}
