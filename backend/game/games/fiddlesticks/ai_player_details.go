package fiddlesticks

type PlayerDetails struct {
	missing_suits [5]bool
}

func createPlayerDetails() *PlayerDetails {
	return &PlayerDetails{}
}

func (d *PlayerDetails) dealHand() {
	d.missing_suits = [5]bool{}
}

func (d *PlayerDetails) missingSuit(suit uint8) {
	d.missing_suits[suit] = true
}
