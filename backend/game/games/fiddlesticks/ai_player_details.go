package fiddlesticks

type PlayerDetails struct {
	missing_suits map[uint8]bool
}

func createPlayerDetails() *PlayerDetails {
	return &PlayerDetails{}
}

func (d *PlayerDetails) dealHand() {
	d.missing_suits = make(map[uint8]bool, 3)
}

func (d *PlayerDetails) missingSuit(suit uint8) {
	d.missing_suits[suit] = true
}
