package goldset

import (
	"fmt"
	"path/filepath"
	"slices"
	"time"
)

func (s *Store) SaveLabel(id string, label Label) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, exists := s.items[id]
	if !exists {
		return fmt.Errorf("item not found")
	}
	if !slices.Contains(item.Legal, label.Action) {
		return fmt.Errorf("choose a legal action")
	}
	if label.Confidence < 0 || label.Confidence > 1 {
		return fmt.Errorf("confidence must be between zero and one")
	}
	actions, err := acceptableActions(item, label.Action, label.Acceptable)
	if err != nil {
		return err
	}
	label.Acceptable = actions
	label.ItemID, label.SavedAt = id, time.Now().UTC().Format(time.RFC3339Nano)
	reference, err := freezeReference(item, "item-snapshot")
	if err != nil {
		return err
	}
	label.Reference = reference
	labels := append(append([]Label(nil), s.labels...), label)
	if err := saveJSON(filepath.Join(s.Directory, "labels.json"), labels); err != nil {
		return err
	}
	s.labels = labels
	s.latest[id] = label
	return nil
}
