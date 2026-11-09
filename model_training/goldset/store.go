package goldset

import (
	"os"
	"path/filepath"
	"sync"
)

type Store struct {
	mu        sync.RWMutex
	Directory string
	items     map[string]Item
	order     []string
	labels    []Label
	latest    map[string]Label
}

func OpenStore(directory string) (*Store, error) {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, err
	}
	s := &Store{Directory: directory, items: map[string]Item{}, latest: map[string]Label{}}
	var items []Item
	if err := loadJSON(filepath.Join(directory, "items.json"), &items); err != nil {
		return nil, err
	}
	if err := loadJSON(filepath.Join(directory, "labels.json"), &s.labels); err != nil {
		return nil, err
	}
	for _, item := range items {
		s.items[item.ID] = item
		s.order = append(s.order, item.ID)
	}
	if err := s.backfillReferences(); err != nil {
		return nil, err
	}
	if err := s.backfillAcceptableActions(); err != nil {
		return nil, err
	}
	for _, label := range s.labels {
		s.latest[label.ItemID] = label
	}
	return s, nil
}
