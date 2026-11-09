package goldset

import (
	"fmt"
	"os"
	"path/filepath"
)

func (s *Store) backfillReferences() error {
	original := append([]Label(nil), s.labels...)
	changed := false
	for i := range s.labels {
		if s.labels[i].Reference != nil {
			continue
		}
		item, exists := s.items[s.labels[i].ItemID]
		if !exists {
			return fmt.Errorf("missing original item for label %s", s.labels[i].ItemID)
		}
		reference, err := freezeReference(item, "stored-item-backfill")
		if err != nil {
			return err
		}
		s.labels[i].Reference = reference
		changed = true
	}
	if !changed {
		return nil
	}
	backup := filepath.Join(s.Directory, "labels.before-reference.json")
	if _, err := os.Stat(backup); os.IsNotExist(err) {
		if err := saveJSON(backup, original); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return saveJSON(filepath.Join(s.Directory, "labels.json"), s.labels)
}
