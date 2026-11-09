package rl

import (
	"reflect"
	"testing"
)

func TestFullScheduleAndCoreWeight(t *testing.T) {
	cases := trainingSchedule(400, 42, 0.9)
	counts := map[int]int{}
	for _, s := range cases {
		counts[s.Settings.Players]++
		if int(s.Settings.MaxRound) != 51/s.Settings.Players {
			t.Fatal("missing legal peak", s)
		}
	}
	for p := 2; p <= 10; p++ {
		want := 10
		if p >= 4 && p <= 8 {
			want = 72
		}
		if counts[p] != want {
			t.Fatal("wrong training mix", counts)
		}
	}
	if !reflect.DeepEqual(cases, trainingSchedule(400, 42, 0.9)) {
		t.Fatal("schedule is not reproducible")
	}
	counts = map[int]int{}
	for _, s := range schedule(360, 43) {
		counts[s.Settings.Players]++
	}
	for p := 2; p <= 10; p++ {
		if counts[p] != 40 {
			t.Fatal("unbalanced evaluation", counts)
		}
	}
	configs := trials(Options{Sweep: true, Hidden: 4, LinearControls: true})
	hidden, linear := 0, 0
	for _, t := range configs {
		if t.Hidden == 4 {
			hidden++
		} else {
			linear++
		}
	}
	if hidden != 24 || linear != 3 {
		t.Fatal("wrong architecture sweep", hidden, linear)
	}
}
