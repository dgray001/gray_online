package rl

import (
	"reflect"
	"testing"
)

func TestParallelEvaluationHasIndependentRandomness(t *testing.T) {
	network := initialNetwork(trial("xavier", 0.002, 0, 1))
	cases := schedule(4, 987)
	channel := make(chan []Episode, 2)
	for i := 0; i < 2; i++ {
		go func() { _, games := evaluate("test", network, cases); channel <- games }()
	}
	if !reflect.DeepEqual(<-channel, <-channel) {
		t.Fatal("parallel games shared action or deal randomness")
	}
}
