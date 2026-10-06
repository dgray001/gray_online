package bridge

import (
	"testing"
	"time"
)

func receive[T any](t *testing.T, channel <-chan T) T {
	t.Helper()
	select {
	case value := <-channel:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for runner")
	}
	var zero T
	return zero
}
