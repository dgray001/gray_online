package goldset

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
)

func AddFoundationBatch(store *Store) (int, error) {
	cases := append(append(foundationChecks(), foundationBids()...), foundationCards()...)
	if len(cases) != 30 {
		return 0, fmt.Errorf("foundation batch must contain 30 items")
	}
	rng := rand.New(rand.NewSource(20261110))
	rng.Shuffle(len(cases), func(i int, j int) { cases[i], cases[j] = cases[j], cases[i] })
	items, checks := []Item{}, []FoundationCheck{}
	manifest := map[string]string{}
	for i, spec := range cases {
		item, err := foundationItem(spec, 900002001+int64(i), i+1)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", spec.Name, err)
		}
		key, err := foundationKey(spec, item)
		if err != nil {
			return 0, err
		}
		if key != nil {
			checks = append(checks, *key)
		}
		items = append(items, item)
		manifest[item.ID] = spec.Name
	}
	if len(checks) != 8 {
		return 0, fmt.Errorf("expected eight independent checks")
	}
	directory := filepath.Join(store.Directory, "batches")
	if err := os.MkdirAll(directory, 0700); err != nil {
		return 0, err
	}
	if err := saveJSON(filepath.Join(directory, "foundation-002-checks.json"), checks); err != nil {
		return 0, err
	}
	if err := saveJSON(filepath.Join(directory, "foundation-002-manifest.json"), manifest); err != nil {
		return 0, err
	}
	return store.Add(items)
}
