package goldset

import "path/filepath"

func (s *Store) Add(items []Item) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all := make([]Item, 0, len(s.order)+len(items))
	seen := map[string]bool{}
	for _, id := range s.order {
		all = append(all, s.items[id])
		seen[id] = true
	}
	added := []Item{}
	for _, item := range items {
		if !seen[item.ID] {
			all = append(all, item)
			added = append(added, item)
			seen[item.ID] = true
		}
	}
	if err := saveJSON(filepath.Join(s.Directory, "items.json"), all); err != nil {
		return 0, err
	}
	for _, item := range added {
		s.items[item.ID] = item
		s.order = append(s.order, item.ID)
	}
	return len(added), nil
}

func (s *Store) Queue() []QueueItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]QueueItem, 0, len(s.order))
	for _, id := range s.order {
		item := s.items[id]
		_, labelled := s.latest[id]
		result = append(result, QueueItem{ID: id, Kind: item.State.Kind, Players: len(item.State.Players), HandSize: item.State.HandSize, Seat: item.State.Seat, Labelled: labelled})
	}
	return result
}

func (s *Store) Get(id string) (Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, exists := s.items[id]
	entry := Entry{Item: item}
	if label, ok := s.latest[id]; ok {
		entry.Label = &label
	}
	return entry, exists
}
