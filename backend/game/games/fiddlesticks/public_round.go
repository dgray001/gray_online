package fiddlesticks

func (f *GameFiddlesticks) observeCompletedRound() {
	f.ensureProfiles()
	for seat, player := range f.players {
		p := &f.public_profiles[seat]
		p.Hands++
		p.TrickFraction += float64(player.tricks) / float64(f.round)
		if player.bet == player.tricks {
			p.Exact++
		}
	}
}

func (f *GameFiddlesticks) SetPriorProfiles(profiles []PublicProfile) {
	f.prior_profiles = append([]PublicProfile(nil), profiles...)
}
