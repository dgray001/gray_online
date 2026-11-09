package goldset

import (
	"os"
	"path/filepath"
	"reflect"
)

func (s *Store) backfillAcceptableActions() error {
	original := append([]Label(nil), s.labels...)
	backup := filepath.Join(s.Directory, "labels.before-equivalences.json")
	corrections := map[string][]int{}
	if _, err := os.Stat(backup); os.IsNotExist(err) {
		if err := loadJSON(filepath.Join(s.Directory, "equivalences.json"), &corrections); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := s.normalizeAcceptable(corrections); err != nil {
		return err
	}
	if reflect.DeepEqual(original, s.labels) {
		return nil
	}
	if _, err := os.Stat(backup); os.IsNotExist(err) {
		if err := saveJSON(backup, original); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return saveJSON(filepath.Join(s.Directory, "labels.json"), s.labels)
}
