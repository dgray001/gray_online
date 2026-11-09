package rl

import "testing"

func TestSixWidthSweepKeepsAllConfigurations(t *testing.T) {
	widths, err := parseWidths("8,16,32,64,128,256")
	if err != nil {
		t.Fatal(err)
	}
	configs := trials(Options{Widths: widths, LinearControls: true})
	if len(configs) != 147 {
		t.Fatal("missing trial configurations", len(configs))
	}
	counts, ids := map[int]int{}, map[string]bool{}
	for _, config := range configs {
		counts[config.Hidden]++
		if ids[config.ID] {
			t.Fatal("duplicate trial", config.ID)
		}
		ids[config.ID] = true
	}
	for _, width := range widths {
		if counts[width] != 24 {
			t.Fatal("wrong number of seeds or recipes", width, counts[width])
		}
	}
	if counts[0] != 3 {
		t.Fatal("linear controls repeated")
	}
	if _, err := parseWidths("8,8"); err == nil {
		t.Fatal("duplicate widths accepted")
	}
}
