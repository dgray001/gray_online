package fiddlesticks

import "github.com/dgray001/gray_online/game/game_utils"

func (f *GameFiddlesticks) ensureProfiles() {
	if len(f.known_voids) != len(f.players) {
		f.known_voids = make([][4]bool, len(f.players))
	}
	if len(f.public_profiles) != len(f.players) {
		f.public_profiles = make([]PublicProfile, len(f.players))
	}
}

func (f *GameFiddlesticks) observeBid(seat int, bid uint8) {
	f.ensureProfiles()
	p := &f.public_profiles[seat]
	p.Bids++
	p.BidFraction += float64(bid) / float64(f.round)
	f.public_history = append(f.public_history, PublicAction{Seat: seat, RoundIndex: f.progress().Index, HandSize: f.round, Kind: "bet", Bid: bid})
}

func (f *GameFiddlesticks) observeCard(seat int, card *game_utils.StandardCard) {
	f.ensureProfiles()
	p := &f.public_profiles[seat]
	p.Cards++
	if card.GetSuit() == f.trump.GetSuit() {
		p.TrumpCards++
	}
	if card.GetNumber() >= 11 {
		p.HighCards++
	}
	if len(f.trick) == 0 {
		p.Leads++
		if card.GetSuit() == f.trump.GetSuit() {
			p.TrumpLeads++
		}
	} else if card.GetSuit() != f.trick[0].GetSuit() {
		f.known_voids[seat][f.trick[0].GetSuit()-1] = true
	}
	a := PublicAction{Seat: seat, RoundIndex: f.progress().Index, HandSize: f.round, Trick: len(f.players[seat].cards_played) + 1, Order: len(f.trick), Kind: "play-card", Suit: card.GetSuit(), Number: card.GetNumber()}
	f.public_history = append(f.public_history, a)
	f.round_history = append(f.round_history, a)
}
