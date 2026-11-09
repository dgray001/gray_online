package fiddlesticks

type RoundOutcome struct {
	Seat   int
	Round  uint8
	Bid    uint8
	Tricks uint8
	Points uint32
}

func (f *GameFiddlesticks) SetRoundObserver(observer func([]RoundOutcome)) {
	f.round_observer = observer
}

func (f *GameFiddlesticks) notifyRoundOutcome() {
	f.observeCompletedRound()
	if f.round_observer == nil {
		return
	}
	outcomes := make([]RoundOutcome, len(f.players))
	for seat, p := range f.players {
		outcomes[seat] = RoundOutcome{Seat: seat, Round: f.round, Bid: p.bet, Tricks: p.tricks}
		if p.bet == p.tricks {
			outcomes[seat].Points = uint32(f.round_points) + uint32(f.trick_points)*uint32(p.bet)
		}
	}
	f.round_observer(outcomes)
}
