package orders

import (
	"reflect"
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"github.com/gin-gonic/gin"
)

// Player 0 holds a redoubt and a soldier at home plus an outpost and a soldier far away; player 1's villager watches home
func notificationGame(t *testing.T, spy bool) *harness.Game {
	t.Helper()
	doc := `{"board_size":3,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":23,"player":0},"units":[{"id":11,"player":0,"count":1},{"id":1,"player":1,"count":1}]}]},{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":21,"player":0},"units":[{"id":11,"player":0,"count":1}]}]}]}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"notifications": doc})
	watcher := defs.UnitConfigs[1]
	watcher.Vision.Space = defs.VisibilityGood
	if spy {
		watcher.Vision.Space = defs.VisibilitySpy
	}
	defs.UnitConfigs[1] = watcher
	return harness.NewConnectedGame(t, "custom:notifications", 1, 2)
}

func playersOf(update *game.UpdateMessage) any {
	return update.Content["game"].(gin.H)["players"]
}

// Behavior notifications reach the owner in full and an observer only for the subjects the observer can spy on
func checkBehaviorNotifications(t *testing.T, g *harness.Game, spy bool) {
	t.Helper()
	p, other := g.Human(0), g.Human(1)
	g.Submit(p)
	g.Action(p, "unsubmit-orders", nil)
	var visible, hidden uint64
	for _, u := range g.Self(p).Units {
		if u.UnitID == 11 && u.Space.X == 0 {
			visible = u.InternalID
		} else if u.UnitID == 11 {
			hidden = u.InternalID
		}
	}
	for _, kind := range []string{"unit", "building"} {
		if kind == "building" {
			visible, hidden = buildingID(g, p, 23), buildingID(g, p, 21)
		}
		ids := []uint64{visible, hidden}
		rawAction(g, p, "set-"+kind+"-behavior", gin.H{"internal_ids": append(append([]uint64(nil), ids...), 999999), "stance": 2, "auto_attack": false, "interrupt_current": true, "target_priority": []uint8{0, 3, 255, 2}})
		owner := nextUpdate(t, g.Base.Players[uint64(p+1)].SettingUpdates)
		if owner.Kind != kind+"-behavior-set" || !reflect.DeepEqual(owner.Content["internal_ids"], ids) || !reflect.DeepEqual(owner.Content["target_priority"], []int{3, 2}) {
			t.Fatalf("owner notification: %+v", owner)
		}
		updates := g.Base.Players[uint64(other+1)].SettingUpdates
		if !spy {
			if len(updates) != 0 {
				t.Fatal("ordinary vision received private behavior notification")
			}
			continue
		}
		observer := nextUpdate(t, updates)
		if observer.Kind != owner.Kind || !reflect.DeepEqual(observer.Content["internal_ids"], []uint64{visible}) || !reflect.DeepEqual(observer.Content["target_priority"], []int{3, 2}) {
			t.Fatalf("observer notification leaked subjects: %+v", observer)
		}
	}
}

func TestBehaviorNotificationsHideFromOrdinaryVision(t *testing.T) {
	checkBehaviorNotifications(t, notificationGame(t, false), false)
}

func TestBehaviorNotificationsAndDefaults(t *testing.T) {
	g := notificationGame(t, true)
	checkBehaviorNotifications(t, g, true)
	p, spy := g.Human(0), g.Human(1)
	ids := unitIDs(g, p, 11)

	// an invalid stance changes nothing, and what each recipient is told matches the authoritative units
	for _, stance := range []uint8{0, uint8(defs.UnitStance_END), 255} {
		g.Action(p, "set-unit-behavior", gin.H{"internal_ids": ids[:1], "stance": 2})
		g.Action(p, "set-unit-behavior", gin.H{"internal_ids": ids[1:], "stance": 4})
		stances := map[uint64]string{ids[0]: encoded(t, entity(g, p, "units", ids[0])["stance"]), ids[1]: encoded(t, entity(g, p, "units", ids[1])["stance"])}
		rawAction(g, p, "set-unit-behavior", gin.H{"internal_ids": ids, "stance": stance, "attack_back": false})
		for _, recipient := range []int{p, spy} {
			update := nextUpdate(t, g.Base.Players[uint64(recipient+1)].SettingUpdates)
			affected := update.Content["internal_ids"].([]uint64)
			if update.Kind != "unit-behavior-set" || (recipient == p && len(affected) != 2) || (recipient == spy && len(affected) != 1) {
				t.Fatalf("invalid stance notification recipients: %+v", update)
			}
			for _, id := range affected {
				unit := entity(g, p, "units", id)
				if encoded(t, unit["stance"]) != stances[id] {
					t.Fatal("invalid stance changed authoritative state")
				}
				if value, exists := update.Content["stance"]; exists && encoded(t, value) != encoded(t, unit["stance"]) {
					t.Fatalf("stance acknowledgement %v differs from unit %v", value, unit["stance"])
				}
			}
			if update.Content["game"] != nil {
				t.Fatal("behavior notification carried a full snapshot")
			}
		}
	}

	g.Action(p, "set-default-unit-stance", gin.H{"stance": uint8(defs.UnitStance_AGGRESSIVE)})
	for _, stance := range []any{nil, 0, uint8(defs.UnitStance_END), 255, -1, "aggressive"} {
		rawAction(g, p, "set-default-unit-stance", gin.H{"stance": stance})
		if update := nextUpdate(t, g.Base.Players[uint64(p+1)].FailedUpdates); update.Kind != "set-default-unit-stance-failed" {
			t.Fatalf("invalid stance accepted: %+v", update)
		}
		if g.Risq.PlayerToFrontend(uint64(p+1), g.PlayerID(p))["default_unit_stance"] != defs.UnitStance_AGGRESSIVE {
			t.Fatal("invalid stance changed the default")
		}
	}

	// partial default updates keep the other defaults and never rewrite existing units
	before := encoded(t, entity(g, p, "units", ids[0]))
	g.Action(p, "set-default-unit-behavior", gin.H{"stance": 2, "attack_back": false, "interrupt_current": true, "target_priority": []uint8{1}})
	rawAction(g, p, "set-default-unit-behavior", gin.H{"target_priority": []uint8{}})
	update := latestUpdate(t, g.Base.Players[uint64(p+1)].SettingUpdates)
	if update.Kind != "default-unit-behavior-set" || update.Content["stance"] != defs.UnitStance(2) || update.Content["attack_back"] != false || update.Content["interrupt_current"] != true || len(update.Content["target_priority"].([]int)) != 0 {
		t.Fatalf("partial update lost defaults: %+v", update)
	}
	if encoded(t, entity(g, p, "units", ids[0])) != before {
		t.Fatal("default behavior changed an existing unit")
	}
}

func drain(updates chan *game.UpdateMessage) {
	for len(updates) > 0 {
		<-updates
	}
}

// Acknowledgements are private to the owner, and a replay always restores the newest complete snapshot whatever was dropped
func TestUpdateDelivery(t *testing.T) {
	g := newGame(t, layout{connected: true, viewers: []uint64{90}})
	p, other := g.Human(0), g.Human(1)
	mine := scan(g, p)
	player, rival := g.Base.Players[uint64(p+1)], g.Base.Players[uint64(other+1)]
	center, redoubt := mine.buildings[1], mine.buildings[23]
	rebuilds := 0
	g.Base.RunInRoom = func(task func()) { task() }
	g.Base.CurrentState = func(client uint64, viewer bool) gin.H {
		rebuilds++
		return g.Risq.ToFrontend(client, viewer)
	}

	g.Action(p, "set-auto-renew", gin.H{"building_id": 3, "count": 0})
	rawAction(g, p, "set-auto-renew", gin.H{"building_id": 3, "count": 2})
	update := nextUpdate(t, player.SettingUpdates)
	if update.Kind != "auto-renew-set" || update.Content["building_id"] != uint32(3) || update.Content["count"] != 2 || update.Content["resources"] == nil || update.Content["game"] != nil || len(g.Self(p).ActiveOrders) != 0 {
		t.Fatalf("queue change did not acknowledge immediately: %+v", update)
	}
	if len(rival.Updates) != 0 || len(rival.SettingUpdates) != 0 || len(g.Base.ViewerUpdates) != 0 || len(g.Self(other).AutoRenewals) != 0 {
		t.Fatal("private auto-renew queue was broadcast")
	}
	g.Action(p, "set-auto-renew", gin.H{"building_id": 3, "count": 0})

	g.Action(p, "set-gather-point", gin.H{"building_id": center, "clear": true})
	rawAction(g, p, "set-gather-point", gin.H{"building_id": center, "location_kind": 1, "location_id": harness.SpaceKey(3, 0)})
	update = nextUpdate(t, player.SettingUpdates)
	if update.Kind != "gather-point-set" || update.Content["building_id"] != center || update.Content["game"] != nil || !reflect.DeepEqual(update.Content["gather_point"], entity(g, p, "buildings", center)["gather_point"]) {
		t.Fatalf("gather point acknowledgement differs from state: %+v", update)
	}
	rawAction(g, p, "set-gather-point", gin.H{"building_id": center, "clear": true})
	update = nextUpdate(t, player.SettingUpdates)
	if update.Kind != "gather-point-set" || update.Content["building_id"] != center || update.Content["gather_point"] != nil {
		t.Fatalf("clear acknowledgement retained gather point: %+v", update)
	}
	if len(rival.Updates) != 0 || len(rival.SettingUpdates) != 0 || len(g.Base.ViewerUpdates) != 0 {
		t.Fatal("gather point notification reached another recipient")
	}

	// settings acknowledgements are unsequenced: overflowing them is harmless and they leave no stale snapshot to replay
	for _, last := range []string{"unit", "building", "gather-point"} {
		drain(player.SettingUpdates)
		rawAction(g, other, "submit-orders", nil)
		drain(player.Updates)
		for range cap(player.SettingUpdates) + 2 {
			rawAction(g, p, "set-unit-behavior", gin.H{"internal_ids": mine.infantry, "stance": 1})
		}
		switch last {
		case "unit":
			rawAction(g, p, "set-unit-behavior", gin.H{"internal_ids": mine.infantry, "stance": uint8(defs.UnitStance_STAND_GROUND)})
		case "building":
			rawAction(g, p, "set-building-behavior", gin.H{"internal_ids": []uint64{redoubt}, "interrupt_current": true})
		case "gather-point":
			rawAction(g, p, "set-gather-point", gin.H{"building_id": center, "clear": true})
		}
		if len(player.SettingUpdates) != cap(player.SettingUpdates) {
			t.Fatalf("%s: setting buffer holds %d, want it full", last, len(player.SettingUpdates))
		}
		g.Base.ResendLastUpdate(uint64(p + 1))
		replayed := nextUpdate(t, player.Updates)
		if replayed.Kind != "start-turn" || snapshotJSON(t, playersOf(replayed)) != snapshotJSON(t, g.Risq.PlayersToFrontend(uint64(p+1))) {
			t.Fatalf("%s: replay returned a snapshot older than the setting change", last)
		}
		before := rebuilds
		g.Base.ResendLastUpdate(uint64(p + 1))
		if rebuilds != before {
			t.Fatalf("%s: regenerated snapshot was rebuilt again", last)
		}
		drain(player.Updates)
		g.Action(other, "unsubmit-orders", nil)
	}

	// replay returns the latest snapshot for any requested id, per recipient, and failed actions never replace it
	g.Submit(p)
	rawAction(g, p, "set-unit-behavior", gin.H{})
	if failures := g.Failed(p); len(failures) != 1 {
		t.Fatal("setup did not reject submitted action")
	}
	drain(player.Updates)
	g.Base.ResendLastUpdate(uint64(p + 1))
	if before := nextUpdate(t, player.Updates); before.Kind != "start-turn" || snapshotJSON(t, playersOf(before)) != snapshotJSON(t, g.Risq.PlayersToFrontend(uint64(p+1))) {
		t.Fatalf("failed action replaced replay or replay missed the submission: %+v", before)
	}
	g.Action(p, "unsubmit-orders", nil)

	// a submission is a small notice to everyone, not a snapshot: the submitter also gets their order list and a stale replay
	drain(player.SettingUpdates)
	drain(rival.SettingUpdates)
	g.Base.ResendLastUpdate(uint64(other + 1))
	drain(player.Updates)
	drain(rival.Updates)
	rawAction(g, p, "submit-orders", nil)
	first := nextUpdate(t, player.SettingUpdates)
	seen := nextUpdate(t, rival.SettingUpdates)
	rawAction(g, p, "unsubmit-orders", nil)
	last := nextUpdate(t, player.SettingUpdates)
	if first.Kind != "submitted-orders" || last.Kind != "unsubmitted-orders" || first.Content["orders_submitted"] != true || last.Content["orders_submitted"] != false {
		t.Fatalf("submission notices have wrong kind or flag: %+v %+v", first, last)
	}
	if first.Content["game"] != nil || first.Content["active_orders"] == nil || first.Content["turn_number"] != g.Risq.TurnNumber() || first.Content["giving_orders"] != true {
		t.Fatalf("owner notice is not a small notice with their orders: %+v", first)
	}
	if seen.Content["player_id"] != g.PlayerID(p) || seen.Content["orders_submitted"] != true || seen.Content["active_orders"] != nil || seen.Content["game"] != nil {
		t.Fatalf("other players learned more than the submission: %+v", seen)
	}
	if len(player.Updates) != 0 || len(rival.Updates) != 0 || len(g.Base.ViewerUpdates) != 0 {
		t.Fatal("a submission built a snapshot")
	}
	if _, stale, _ := g.Base.LatestUpdate(uint64(p + 1)); !stale {
		t.Fatal("the submitter's stored snapshot is not marked stale")
	}
	if _, stale, _ := g.Base.LatestUpdate(uint64(other + 1)); stale {
		t.Fatal("another player's snapshot was invalidated by a submission")
	}
	for _, requested := range []int{-9, 1, 99} {
		g.Base.ResendPlayerUpdate(uint64(p+1), requested)
		if got := nextUpdate(t, player.Updates); got.Kind != "start-turn" || snapshotJSON(t, playersOf(got)) != snapshotJSON(t, g.Risq.PlayersToFrontend(uint64(p+1))) {
			t.Fatalf("replay %d did not return the current snapshot: %+v", requested, got)
		}
	}
	g.Base.ResendLastUpdate(uint64(other + 1))
	if got := nextUpdate(t, rival.Updates); got.Kind != "start-turn" {
		t.Fatal("recipient histories were mixed")
	}

	// viewers replay the latest turn snapshot too, and a new turn replaces the history
	viewer := g.Base.Viewers[90]
	drain(g.Base.ViewerUpdates)
	g.Base.ResendLastUpdate(90)
	turnViewer := nextUpdate(t, viewer.Updates)
	rawAction(g, p, "submit-orders", nil)
	rawAction(g, p, "unsubmit-orders", nil)
	if len(g.Base.ViewerUpdates) != 0 || len(viewer.Updates) != 0 {
		t.Fatal("viewers were sent submission updates")
	}
	for _, requested := range []int{-9, 1, 99} {
		g.Base.ResendViewerUpdate(90, requested)
		if got := nextUpdate(t, viewer.Updates); !reflect.DeepEqual(got, turnViewer) {
			t.Fatal("viewer replay did not return the turn snapshot")
		}
	}
	g.Base.ResendLastUpdate(90)
	if got := nextUpdate(t, viewer.Updates); !reflect.DeepEqual(got, turnViewer) {
		t.Fatal("last viewer replay differs")
	}
	g.EndTurn()
	g.Base.ResendViewerUpdate(90, 1)
	if got := nextUpdate(t, viewer.Updates); got.Id <= turnViewer.Id || got.Kind != "start-turn" {
		t.Fatal("new turn did not replace viewer history")
	}
	g.Base.ResendPlayerUpdate(999999, 1)
	g.Base.ResendViewerUpdate(999999, 1)
	g.Base.ResendLastUpdate(999999)
	if len(viewer.Updates) != 0 {
		t.Fatal("unknown recipient received replay")
	}
}

func TestLatestUpdateLookup(t *testing.T) {
	g := newGame(t, layout{connected: true, viewers: []uint64{90}})
	p := g.Human(0)
	client := uint64(p + 1)
	player := g.Base.Players[client]
	start, stale, ok := g.Base.LatestUpdate(client)
	if !ok || stale || start < 1 {
		t.Fatalf("lookup gave id %d stale %v ok %v, want a fresh positive id", start, stale, ok)
	}
	rawAction(g, p, "set-unit-behavior", gin.H{"internal_ids": scan(g, p).infantry, "stance": 1})
	if id, stale, _ := g.Base.LatestUpdate(client); id != start || !stale {
		t.Fatalf("setting change gave id %d stale %v, want %d stale", id, stale, start)
	}
	drain(player.Updates)
	drain(g.Base.ViewerUpdates)
	rawAction(g, p, "submit-orders", nil)
	if id, stale, _ := g.Base.LatestUpdate(client); id != start || !stale || len(player.Updates) != 0 {
		t.Fatalf("a submission gave id %d stale %v, want the same id %d stale and no snapshot", id, stale, start)
	}
	g.EndTurn()
	if id, stale, _ := g.Base.LatestUpdate(client); id <= start || stale {
		t.Fatalf("new turn gave id %d stale %v, want an id above %d and fresh", id, stale, start)
	}
	if id, stale, ok := g.Base.LatestUpdate(90); !ok || stale || id < 2 {
		t.Fatalf("viewer lookup gave id %d stale %v ok %v, want a fresh id from the new turn", id, stale, ok)
	}
	if _, _, ok := g.Base.LatestUpdate(999999); ok {
		t.Fatal("unknown client was found")
	}
}

// The last submission resolves the turn at once, so its notice must name the turn it was sent in for clients to discard it late
func TestLastSubmissionNoticeIsStampedWithItsOwnTurn(t *testing.T) {
	g := newGame(t, layout{connected: true})
	p, other := g.Human(0), g.Human(1)
	player := g.Base.Players[uint64(p+1)]
	turn := g.Risq.TurnNumber()
	drain(player.SettingUpdates)
	rawAction(g, other, "submit-orders", nil)
	rawAction(g, p, "submit-orders", nil)
	if g.Risq.TurnNumber() == turn {
		t.Fatal("the last submission did not resolve the turn")
	}
	own := latestUpdate(t, player.SettingUpdates)
	if own.Kind != "submitted-orders" || own.Content["turn_number"] != turn || own.Content["giving_orders"] != false {
		t.Fatalf("last notice %+v, want turn %d with orders no longer being given", own.Content, turn)
	}
}
