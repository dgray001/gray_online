package risq_mapgen_tests

import "testing"

func createWithAis(count int) error {
	_, err := engineStarts("script:ring", count, 1)
	return err
}

func TestCreateGameRejectsInvalidPlayerCounts(t *testing.T) {
	for _, count := range []int{0, 1, 13} {
		if err := createWithAis(count); err == nil {
			t.Errorf("%d players should be rejected", count)
		}
	}
	for _, count := range []int{2, 12} {
		if err := createWithAis(count); err != nil {
			t.Errorf("%d players should be accepted: %v", count, err)
		}
	}
}
