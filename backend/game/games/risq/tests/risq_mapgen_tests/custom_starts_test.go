package risq_mapgen_tests

import "testing"

const startTemplate = `"player_start":{"size":0,"terrain":1,"spaces":[{"x":0,"y":0,"zones":[` +
	`{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":1,"player":0,"count":3}]}]}]}`

func startsDoc(player_start string, spaces string) string {
	return `{"board_size":2,"players":2,` + player_start + `,"spaces":[` + spaces + `]}`
}

const twoSlots = `{"x":-2,"y":0,"player_slot":0},{"x":2,"y":0,"player_slot":1},{"x":0,"y":0}`

func TestPlayerStartStampsEachSlotForItsOwnPlayer(t *testing.T) {
	board := mustGenerate(t, startsDoc(startTemplate, twoSlots), 2)
	for player, x := range map[int]int{0: -2, 1: 2} {
		center := zoneAt(t, board, x, 0, 0, 0)
		if center.Building.ID != 1 || center.Building.Player != player || len(center.Units) != 3 || center.Units[0].Player != player {
			t.Errorf("slot %d center = %+v, want its own building and 3 units", player, center)
		}
	}
	if mid := zoneAt(t, board, 0, 0, 0, 0); mid.Building.ID != 0 || len(mid.Units) != 0 {
		t.Errorf("unslotted space was stamped: %+v", mid)
	}
}

func TestPlayerStartOnlyStampsPlayingSlots(t *testing.T) {
	board := mustGenerate(t, startsDoc(startTemplate, twoSlots), 1)
	if len(board.PlayerSpaces()) != 1 || zoneAt(t, board, 2, 0, 0, 0).Building.ID != 0 {
		t.Errorf("one player in the game but players placed: %v", board.PlayerSpaces())
	}
}

func TestPlayerStartResetsHomeTerrainAndOccupants(t *testing.T) {
	spaces := `{"x":-2,"y":0,"player_slot":0,"terrain":15,"zones":[{"x":1,"y":0,"resource":41}]},{"x":2,"y":0,"player_slot":1}`
	board := mustGenerate(t, startsDoc(startTemplate, spaces), 2)
	home := board.Inspect()[0]
	if home.Terrain != 1 || zoneAt(t, board, -2, 0, 1, 0).Resource != 0 {
		t.Errorf("home terrain %d and edge resource %d, want terrain 1 and the resource cleared", home.Terrain, zoneAt(t, board, -2, 0, 1, 0).Resource)
	}
}

func TestPlayerStartRejectsBadSlots(t *testing.T) {
	slot := func(spaces string) string { return startsDoc(startTemplate, spaces) }
	mustFail(t, slot(`{"x":-2,"y":0,"player_slot":0},{"x":2,"y":0,"player_slot":0}`), 2, "invalid or duplicate player slot")
	mustFail(t, slot(`{"x":-2,"y":0,"player_slot":0},{"x":2,"y":0,"player_slot":2}`), 2, "invalid or duplicate player slot")
	mustFail(t, slot(`{"x":-2,"y":0,"player_slot":0},{"x":2,"y":0}`), 2, "missing player slot 1")
}

func TestPlayerStartRejectsBadTemplates(t *testing.T) {
	template := func(body string) string { return `"player_start":{` + body + `}` }
	mustFail(t, startsDoc(template(`"size":9,"spaces":[]`), twoSlots), 2, "invalid player start size")
	mustFail(t, startsDoc(template(`"size":0,"spaces":[{"x":0,"y":0,"player_slot":0}]`), twoSlots), 2, "invalid player start template space")
	mustFail(t, startsDoc(template(`"size":0,"spaces":[{"x":1,"y":0}]`), twoSlots), 2, "invalid player start template space")
	mustFail(t, startsDoc(template(`"size":0,"terrain":152,"spaces":[]`), twoSlots), 2, "invalid player start terrain")
}

func TestPlayerStartRejectsOverlapAndClipping(t *testing.T) {
	big := `"player_start":{"size":1,"terrain":1,"spaces":[]}`
	mustFail(t, startsDoc(big, `{"x":-2,"y":0,"player_slot":0},{"x":2,"y":0,"player_slot":1}`), 2, "clipped or impassable footprint")
	board := func(slots map[[2]int]int) string {
		return `{"board_size":3,"players":2,` + big + `,"spaces":[` + hexSpaces(3, slots) + `]}`
	}
	mustFail(t, board(map[[2]int]int{{-1, 0}: 0, {1, 0}: 1}), 2, "overlaps another start area")
	mustFail(t, board(map[[2]int]int{{-1, 0}: 0, {1, -1}: 1}), 2, "overlaps another start area")
	mustGenerate(t, board(map[[2]int]int{{-2, 0}: 0, {1, 0}: 1}), 2)
}
