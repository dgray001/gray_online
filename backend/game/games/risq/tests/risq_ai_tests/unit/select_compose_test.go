package unit

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	. "github.com/dgray001/gray_online/game/games/risq/ai"
)

func at(x int) ZoneRef { return ZoneRef{Space: Coordinate{X: x}} }

func TestRetainedActionsRejectMalformedOptions(t *testing.T) {
	for _, action := range []string{
		`{"action":"set_var","name":"x","value":1,"perssit":true}`,
		`{"action":"set_var","name":"x","value":1,"persist":"true"}`,
		`{"action":"for_each","source":"units","as":"u","do":[],"maax":2}`,
		`{"action":"if","when":{"always":{}},"then":[],"els":[]}`,
		`{"action":"if","when":{"always":{}},"then":[],"else":null}`,
		`{"action":"select","pick":{"kind":"units"},"into":{"set":"s","persist":"true"}}`,
		`{"action":"select","pick":{"kind":"units"},"into":{"set":"s","bucket":2}}`,
		`{"action":"select","pick":{"kind":"units"},"into":{"set":"s","var":false}}`,
		`{"action":"select","pick":{"kind":"units"},"into":{"set":"s","var":""}}`,
		`{"action":"select","pick":null}`,
		`{"action":"select","pick":{"kind":"units"},"actors":null}`,
		`{"action":"select","pick":{"kind":"units"},"assign":null}`,
		`{"action":"select","pick":{"kind":"units"},"issue":null}`,
		`{"action":"select","pick":{"kind":"units"},"into":null}`,
		`{"action":"select","pick":{"kind":"units"},"mode":null}`,
		`{"action":"select","pick":{"kind":"units","where":{"owner":true}}}`,
		`{"action":"select","pick":{"kind":"buildings","where":{"state":true}}}`,
		`{"action":"select","pick":{"kind":"zones","where":{"near":{"unit":false,"of":{"x":0,"y":0}}}}}`,
		`{"action":"select","pick":{"kind":"zones","where":{"near":{"unit":"","of":{"x":0,"y":0}}}}}`,
		`{"action":"select","pick":{"kind":"zones","where":{"near":{"distnace":2,"of":{"x":0,"y":0}}}}}`,
	} {
		if loadError(rule(action)) == nil {
			t.Errorf("malformed options accepted: %s", action)
		}
	}
}

func TestSelectRejectsMixedActorlessVerbsInEitherOrder(t *testing.T) {
	for _, verb := range []string{`{"hire":{"unit_id":2}}`, `{"cancel_foundation":{}}`} {
		for _, verbs := range []string{verb + `,{"move":{}}`, `{"move":{}},` + verb} {
			if loadError(rule(`{"action":"select","pick":{"kind":"zones"},"issue":[`+verbs+`]}`)) == nil {
				t.Errorf("mixed actorless verbs accepted: %s", verbs)
			}
		}
	}
}

func TestSelectControlConfigErrors(t *testing.T) {
	for _, action := range []string{
		`{"action":"select","pick":{"kind":"zones"},"issue":[{"cancel_foundation":{}}]}`,
		`{"action":"select","actors":{"kind":"resources"},"issue":[{"delete":{}}]}`,
		`{"action":"select","actors":{"kind":"units"},"assign":{"capacity":1},"issue":[{"delete":{}}]}`,
		`{"action":"select","actors":{"kind":"buildings"},"mode":"append","issue":[{"delete":{}}]}`,
		`{"action":"select","pick":{"kind":"zones"},"mode":"append","issue":[{"hire":{"unit_id":2}}]}`,
		`{"action":"select","pick":{"kind":"foundations"},"mode":"append","issue":[{"cancel_foundation":{}}]}`,
		`{"action":"select","pick":{"kind":"units"},"mode":"append"}`,
		`{"action":"select","issue":[{"stop":{}}]}`,
		`{"action":"select","actors":{"kind":"zones"},"issue":[{"stop":{}}]}`,
		`{"action":"select","actors":{"kind":"units"},"pick":{"kind":"zones"},"issue":[{"stop":{}}]}`,
		`{"action":"select","actors":{"kind":"units"},"assign":{"capacity":1},"issue":[{"stop":{}}]}`,
		`{"action":"select","actors":{"kind":"units"},"mode":"append","issue":[{"stop":{}}]}`,
		`{"action":"select","actors":{"kind":"units"},"issue":[{"stop":{}},{"move":{}}]}`,
		`{"action":"select","actors":{"kind":"units"},"issue":[{"set_behavior":{}}]}`,
		`{"action":"select","actors":{"kind":"units"},"issue":[{"set_behavior":{"auto_attack":false}}]}`,
		`{"action":"select","actors":{"kind":"buildings"},"issue":[{"set_behavior":{"stance":"passive"}}]}`,
		`{"action":"select","actors":{"kind":"units"},"issue":[{"set_behavior":{"attack_back":"false"}}]}`,
		`{"action":"select","actors":{"kind":"units"},"issue":[{"stop":{"orders":["research"]}}]}`,
		`{"action":"select","actors":{"kind":"units"},"issue":[{"stop":{"item_ids":[2]}}]}`,
	} {
		if err := loadError(rule(action)); err == nil {
			t.Errorf("invalid control accepted: %s", action)
		}
	}
}

func TestBucketSinksRejectPersist(t *testing.T) {
	for _, sink := range []string{`"bucket":"group"`, `"bucket":"all","mode":"remove"`} {
		for _, persist := range []string{"true", "false"} {
			for _, count := range []string{"", `,"var":"count"`} {
				config := rule(`{"action":"select","pick":{"kind":"units"},"into":{` + sink + `,"persist":` + persist + count + `}}`)
				if err := loadError(config); err == nil {
					t.Errorf("bucket sink accepted persist: %s", config)
				}
			}
		}
	}
}

func TestNearAcceptsCoordinateExpressionsAndZoneCoordinates(t *testing.T) {
	view := selectTestView{loopTestView: loopTestView{spaces: lineOfSpaces(0, 0, 0), zones: []ZoneInfo{{Location: at(2)}, {Location: ZoneRef{Space: at(2).Space, Zone: Coordinate{X: 1}}}}}}
	for _, kind := range []string{"spaces", "zones"} {
		origin, unit := `{"x":"var(anchor_x)","y":0}`, "space"
		if kind == "zones" {
			origin, unit = `{"x":"var(anchor_x)","y":0,"zone_x":1,"zone_y":0}`, "zone"
		}
		config := rule(fmt.Sprintf(`{"action":"set_var","name":"anchor_x","value":2},
			{"action":"select","pick":{"kind":%q,"where":{"near":{"distance":0,"unit":%q,"of":%s}}},"into":{"set":"chosen","var":"n"}}`, kind, unit, origin))
		if got := decide(t, config, view, "n")["n"]; got != 1 {
			t.Errorf("%s: selected %v, want exactly the coordinate anchor", kind, got)
		}
	}
}

func TestSelectFailedLimitsDoNotWriteSinks(t *testing.T) {
	view := selectTestView{loopTestView: loopTestView{spaces: lineOfSpaces(1)}}
	for _, key := range []string{"skip", "limit"} {
		config := rule(fmt.Sprintf(`{"action":"set_var","name":"picked","value":9},
			{"action":"select","pick":{"kind":"spaces",%q:"1 / 0"},"into":{"set":"chosen","var":"picked"}}`, key))
		if got := decide(t, config, view, "picked")["picked"]; got != 9 {
			t.Errorf("failed %s overwrote the sink count with %v", key, got)
		}
	}
}

func TestSinkRemoveFromAllBucketsFreesUnitsEverywhere(t *testing.T) {
	units := []UnitView{villager(1, Coordinate{}), villager(2, Coordinate{}), villager(3, Coordinate{})}
	view := selectTestView{loopTestView: loopTestView{units: units}, idle: units, turn: 1}
	into := func(ids string, bucket string, mode string) string {
		return `{"action":"select","pick":{"kind":"units","where":{"in":{"uid":[` + ids + `]}}},"into":{"bucket":"` + bucket + `","mode":"` + mode + `"}}`
	}
	size := func(name string, bucket string) string {
		return `{"action":"set_var","name":"` + name + `","value":{"count":{"kind":"units","where":{"bucket":"` + bucket + `"}}}}`
	}
	config := rule(into("1,2", "a", "replace") + `,` + into("1,3", "b", "replace") + `,` + into("1", "all", "remove") + `,` + size("a_n", "a") + `,` + size("b_n", "b"))
	got := decide(t, config, view, "a_n", "b_n")
	if got["a_n"] != 1 || got["b_n"] != 1 {
		t.Errorf("after freeing unit 1 from every bucket: a has %v, b has %v, want 1 and 1", got["a_n"], got["b_n"])
	}
	for _, mode := range []string{"replace", "add"} {
		if loadError(rule(into("1", "all", mode))) == nil {
			t.Errorf("bucket %q accepted with mode %q", "all", mode)
		}
	}
}

func TestSelectBucketModeConfigErrors(t *testing.T) {
	for _, mode := range []string{`"append"`, `""`, `null`, `true`, `1`, `{}`} {
		config := rule(fmt.Sprintf(`{"action":"select","pick":{"kind":"units"},"into":{"bucket":"group","mode":%s}}`, mode))
		if err := loadError(config); err == nil {
			t.Errorf("invalid bucket mode %s accepted", mode)
		}
	}
	config := rule(`{"action":"select","pick":{"kind":"units"},"into":{"set":"chosen","mode":"replace"}}`)
	if err := loadError(config); err == nil {
		t.Error("bucket mode accepted on a set sink")
	}
}

func TestSelectAssignmentSkipsInvalidBuildSitesWithoutSpending(t *testing.T) {
	worker := villager(1, Coordinate{})
	view := selectTestView{loopTestView: loopTestView{units: []UnitView{worker}, spaces: lineOfSpaces(0, 0), zones: []ZoneInfo{{Location: at(1)}, {Location: at(2)}}}, idle: []UnitView{worker}, blocked_zones: []ZoneRef{at(2)}}
	action := `{"action":"select","pick":{"kind":"zones","capacity":1,"rank":"var(target.x)"},"issue":[{"build":{"building_id":2}}]}`
	orders := selectOrders(t, rule(action), view)
	if len(orders) != 1 || orders[0].TargetID != spaceCode(at(1).Space) || !orders[0].ClearPreviousOrders {
		t.Fatalf("got %+v, want one clearing build order for the valid site", orders)
	}
	config := rule(action + `,{"action":"move","targets":"coordinate","coordinate":{"x":"wood","y":0}}`)
	decision := parseSelectModel(t, config).DecideOrders(decisionView{view})
	if len(decision.Orders) != 2 || decision.Orders[1].TargetID != 90 {
		t.Errorf("got %+v, want one foundation and a remaining-wood probe of 90", decision.Orders)
	}
}

func TestSelectRejectsMismatchedSetKinds(t *testing.T) {
	worker := villager(1, at(1).Space)
	view := selectTestView{loopTestView: loopTestView{units: []UnitView{worker}, buildings: []BuildingView{{InternalID: 1, Location: at(1)}}, spaces: lineOfSpaces(0), zones: []ZoneInfo{{Location: at(1)}}, regions: []RegionInfo{{Spaces: []Coordinate{at(1).Space}}}}, idle: []UnitView{worker}}
	for _, filter := range []string{"in_set", "not_in_set", "region_in_set"} {
		for _, kinds := range [][2]string{{"units", "buildings"}, {"spaces", "zones"}, {"units", "spaces"}} {
			var diagnostics bytes.Buffer
			config := rule(fmt.Sprintf(`{"action":"select","pick":{"kind":%q},"into":{"set":"chosen"}},{"action":"select","pick":{"kind":%q,"where":{%q:"chosen"}},"issue":[{"move":{}}]}`, kinds[0], kinds[1], filter))
			if orders := jsonModel(t, config, &diagnostics).DecideOrders(view).Orders; len(orders) != 0 {
				t.Errorf("%s %v: emitted %+v", filter, kinds, orders)
			}
			if !strings.Contains(diagnostics.String(), `set "chosen" contains`) {
				t.Errorf("%s %v: missing kind diagnostic: %s", filter, kinds, &diagnostics)
			}
		}
	}
}

func TestNearOfAndAtCannotReadTheEnclosingRole(t *testing.T) {
	rejected := map[string]string{
		"near.of": `{"action":"select","pick":{"kind":"zones","where":{"near":{"distance":1,"of":{"kind":"units","where":{"if":"var(target.x)"}}}}},"into":{"set":"x"}}`,
		"at":      `{"action":"select","pick":{"kind":"zones","where":{"at":{"if":"var(target.gold_income)"}}},"into":{"set":"x"}}`,
	}
	for key, action := range rejected {
		var diagnostics bytes.Buffer
		if _, loaded := jsonModel(t, rule(action), &diagnostics).(*RulesModel); loaded || !strings.Contains(diagnostics.String(), "resolved once") {
			t.Errorf("%s reading the enclosing role: loaded=%v, diagnostics %q, want a rejected config with a scope error", key, loaded, diagnostics.String())
		}
	}
	allowed := map[string]string{
		"its own role":        `{"action":"select","pick":{"kind":"zones","where":{"near":{"distance":1,"of":{"kind":"units","where":{"if":"var(anchor.health)"}}}}},"into":{"set":"x"}}`,
		"a per-item let":      `{"action":"select","pick":{"kind":"zones","let":{"hot":{"count":{"kind":"units","where":{"if":"var(item.health) - var(target.x)"}}}},"where":{"if":"var(target.hot)"}},"into":{"set":"x"}}`,
		"a per-item distance": `{"action":"select","pick":{"kind":"zones","where":{"near":{"distance":"var(target.x) + 1","of":{"kind":"units"}}}},"into":{"set":"x"}}`,
	}
	for name, action := range allowed {
		if err := loadError(rule(action)); err != nil {
			t.Errorf("%s rejected: %v", name, err)
		}
	}
}

func TestSelectHasNoHomeBasedFields(t *testing.T) {
	for _, field := range []string{"distance_home", "zone_distance_home", "distance", "zone_distance"} {
		config := rule(`{"action":"select","pick":{"kind":"spaces","rank":"var(target.` + field + `)"},"into":{"set":"x"}}`)
		if loadError(config) == nil {
			t.Errorf("select accepted the home-based field %s", field)
		}
	}
	legacy := rule(`{"action":"for_each","source":"spaces","as":"s","do":[{"action":"set_var","name":"x","value":"var(s.distance_home)"}]}`)
	if err := loadError(legacy); err != nil {
		t.Errorf("legacy for_each lost distance_home: %v", err)
	}
}

func TestNearNestsSelectorsToAnyDepth(t *testing.T) {
	view := selectTestView{
		loopTestView: loopTestView{
			enemy_units: []UnitView{
				{InternalID: 50, PlayerID: 1, Location: at(2)},
				{InternalID: 51, PlayerID: 1, Location: at(9)},
			},
		},
		foundations: []FoundationView{
			{BuildingID: 2, Location: at(1), Planned: true},
			{BuildingID: 2, Location: at(5), Planned: true},
			{BuildingID: 2, Location: at(8), Planned: true},
			{BuildingID: 2, Location: ZoneRef{Space: Coordinate{X: 1}, Zone: Coordinate{X: 1}}},
			{BuildingID: 3, Location: at(2), Planned: true},
		},
		turn: 1,
	}
	config := rule(`{"action":"select","into":{"set":"hot","var":"hot_n"},"pick":{"kind":"foundations","where":{
		"building_ids":[2],"in":{"is_planned":[1]},
		"near":{"distance":2,"of":{"kind":"units","where":{"owner":"enemy",
			"near":{"distance":2,"of":{"kind":"buildings","where":{"owner":"own","in":{"building_id":[1]}}}}}}}}}}`)
	if got := decide(t, config, view, "hot_n")["hot_n"]; got != 1 {
		t.Errorf("planned house foundations near enemies that are near my village center = %v, want only the one at x=1", got)
	}
}

func TestSeveralNearConditionsMustAllHold(t *testing.T) {
	view := selectTestView{
		loopTestView: loopTestView{
			spaces: lineOfSpaces(0, 0, 0, 0, 0),
			enemy_units: []UnitView{
				{InternalID: 50, PlayerID: 1, Location: at(2)}, {InternalID: 51, PlayerID: 1, Location: at(4)},
			},
		},
		turn: 1,
	}
	between := rule(`{"action":"select","into":{"set":"x","var":"n"},"pick":{"kind":"spaces","where":{"near":[
		{"distance":1,"of":{"kind":"units","where":{"owner":"enemy","in":{"uid":[50]}}}},
		{"distance":1,"of":{"kind":"units","where":{"owner":"enemy","in":{"uid":[51]}}}}]}}}`)
	if got := decide(t, between, view, "n")["n"]; got != 1 {
		t.Errorf("spaces within 1 of both enemies = %v, want only the one between them", got)
	}
}

func TestNearMeasuresInSpaceOrZoneSteps(t *testing.T) {
	view := selectTestView{
		loopTestView: loopTestView{spaces: lineOfSpaces(0, 0, 0), enemy_units: []UnitView{{InternalID: 50, PlayerID: 1, Location: at(2)}}},
		turn:         1,
	}
	count := func(distance int, unit string) float64 {
		config := rule(`{"action":"select","into":{"set":"x","var":"n"},"pick":{"kind":"spaces","where":{"near":{"distance":` +
			string(rune('0'+distance)) + `,"unit":"` + unit + `","of":{"kind":"units","where":{"owner":"enemy"}}}}}}`)
		return decide(t, config, view, "n")["n"]
	}
	if count(1, "space") != 3 || count(5, "zone") != 1 {
		t.Errorf("1 space step reached %v spaces and 5 zone steps reached %v, want 3 and 1", count(1, "space"), count(5, "zone"))
	}
}

func TestInFiltersTakeNamesForAnyEnumField(t *testing.T) {
	gathering := &CurrentOrder{Kind: OrderKindGather, TargetResource: &ResourceView{InternalID: 5}}
	moving := &CurrentOrder{Kind: OrderKindMove, TargetSpace: &Coordinate{X: 3}}
	attacking := &CurrentOrder{Kind: OrderKindAttackSpace, TargetSpace: &Coordinate{X: 3}}
	view := selectTestView{
		loopTestView: loopTestView{units: []UnitView{
			{InternalID: 1, Type: UnitTypeEconomic, CurrentOrder: gathering},
			{InternalID: 2, Type: UnitTypeInfantry, Kind: UnitMilitary, CurrentOrder: moving},
			{InternalID: 3, Type: UnitTypeArcher, Kind: UnitMilitary, CurrentOrder: attacking},
			{InternalID: 4, Type: UnitTypeEconomic},
		}},
		turn: 1,
	}
	config := rule(`{"action":"set_var","name":"eligible","value":{"count":{"kind":"units","where":{"in":{"order_kind":["none","gather","move"]}}}}},
		{"action":"set_var","name":"military","value":{"count":{"kind":"units","where":{"in":{"unit_type":["military"]}}}}},
		{"action":"set_var","name":"by_id","value":{"count":{"kind":"units","where":{"in":{"uid":[1,4]}}}}}`)
	vars := decide(t, config, view, "eligible", "military", "by_id")
	if vars["eligible"] != 3 || vars["military"] != 2 || vars["by_id"] != 2 {
		t.Errorf("eligible %v, military %v, by id %v; want 3, 2, 2", vars["eligible"], vars["military"], vars["by_id"])
	}
	for name, where := range map[string]string{
		"unknown name":  `{"in":{"order_kind":["teleport"]}}`,
		"unknown field": `{"in":{"nope":[1]}}`,
		"empty list":    `{"in":{"unit_type":[]}}`,
		"not an object": `{"in":["gather"]}`,
	} {
		if loadError(rule(`{"action":"set_var","name":"x","value":{"count":{"kind":"units","where":`+where+`}}}`)) == nil {
			t.Errorf("%s: malformed in filter accepted", name)
		}
	}
}

func TestPrependRestoresTasksAfterAllAssignmentRounds(t *testing.T) {
	view := tieView()
	view.units = append(view.units, villager(2, Coordinate{}))
	for i := range view.units {
		view.units[i].CurrentOrder = &CurrentOrder{Kind: OrderKindGather, TargetResource: &ResourceView{InternalID: uint64(5 + i)}}
	}
	config := rule(`{"action":"select","mode":"prepend","actors":{"kind":"units","rank":"var(actor.uid)"},"pick":{"kind":"spaces","rank":"var(target.x)","limit":1},"issue":[{"move":{}}]}`)
	orders := selectOrders(t, config, view)
	if len(orders) != 4 {
		t.Fatalf("orders = %+v, want two moves then two restored tasks", orders)
	}
	for i, target := range []int64{1200, 1200, 6, 5} {
		kind := kindMoveSpace
		if i >= 2 {
			kind = kindGather
		}
		if orders[i].TargetID != target || orders[i].OrderType != kind || orders[i].Subjects[0] != uint64(2-i%2) {
			t.Errorf("order %d = %+v, want target %d in per-actor prepend order", i, orders[i], target)
		}
	}
}

func TestQueueModesAppendAndPrependOrders(t *testing.T) {
	job := &CurrentOrder{Kind: OrderKindGather, TargetResource: &ResourceView{InternalID: 5}}
	worker := UnitView{InternalID: 1, Type: UnitTypeEconomic, CurrentOrder: job, Builds: []Producible{{ID: 2}}}
	view := selectTestView{loopTestView: loopTestView{units: []UnitView{worker}, spaces: lineOfSpaces(0)}, idle: []UnitView{worker}, turn: 1}
	run := func(queue string) []Order {
		return selectOrders(t, rule(`{"action":"select","mode":`+queue+`,"pick":{"kind":"spaces","limit":1},"issue":[{"move":{}}]}`), view)
	}
	if got := run(`"replace"`); len(got) != 1 || !got[0].ClearPreviousOrders {
		t.Errorf("replace orders = %+v, want one clearing move", got)
	}
	if got := run(`"append"`); len(got) != 1 || got[0].ClearPreviousOrders {
		t.Errorf("append orders = %+v, want one queued move", got)
	}
	got := run(`"prepend"`)
	if len(got) != 2 || got[0].OrderType != kindMoveSpace || !got[0].ClearPreviousOrders || got[1].OrderType != kindGather || got[1].TargetID != 5 || got[1].ClearPreviousOrders {
		t.Errorf("prepend orders = %+v, want a clearing move followed by the old gather queued behind it", got)
	}
	if loadError(rule(`{"action":"select","mode":"front","pick":{"kind":"spaces"},"issue":[{"move":{}}]}`)) == nil {
		t.Error("unknown queue mode accepted")
	}
}

func TestBuildableZonesDerivedInJSON(t *testing.T) {
	zone := func(space int, x int, y int, building int, resource bool) ZoneInfo {
		return ZoneInfo{Location: ZoneRef{Space: Coordinate{X: space}, Zone: Coordinate{X: x, Y: y}}, BuildingPlayer: building, HasResource: resource}
	}
	view := selectTestView{
		loopTestView: loopTestView{
			spaces: []SpaceInfo{
				{Space: Coordinate{X: 1}, Owner: 0, Vision: 3}, {Space: Coordinate{X: 2}, Owner: -1, Vision: 3},
				{Space: Coordinate{X: 3}, Owner: 1, Vision: 3}, {Space: Coordinate{X: 7}, Owner: -1, Vision: 3},
			},
			zones: []ZoneInfo{
				zone(1, 0, 0, -1, false), zone(1, 1, 0, 0, false), zone(1, 0, 1, -1, false), zone(1, -1, 0, -1, true),
				zone(2, 0, 0, -1, false), zone(7, 0, 0, -1, false), zone(3, 0, 0, -1, false),
			},
		},
		foundations: []FoundationView{{BuildingID: 2, Location: ZoneRef{Space: Coordinate{X: 1}, Zone: Coordinate{Y: 1}}, Planned: true}},
		turn:        1,
	}
	config := rule(`{"action":"select","into":{"set":"sites","var":"site_n"},"pick":{"kind":"zones",
		"let":{
			"foundations_here":{"count":{"kind":"foundations","where":{"if":"1 - var(item.zone_distance_to_target)"}}},
			"owned_within_one":{"count":{"kind":"spaces","where":{"owner":"own","if":"2 - var(item.distance_to_target)"}}}},
		"where":{"if":["0 - var(target.building_player)","1 - var(target.has_resource)","1 - var(target.foundations_here)",
			"var(target.owned_within_one)","1 - var(target.space_owner)"]}}},
		{"action":"set_var","name":"in_owned","value":{"count":{"kind":"zones","where":{"in_set":"sites","in":{"space_owner":[0]}}}}},
		{"action":"set_var","name":"in_unowned","value":{"count":{"kind":"zones","where":{"in_set":"sites","in":{"space_owner":[-1]}}}}}`)
	vars := decide(t, config, view, "site_n", "in_owned", "in_unowned")
	if vars["site_n"] != 2 || vars["in_owned"] != 1 || vars["in_unowned"] != 1 {
		t.Errorf("buildable zones %v (in my space %v, in an adjacent unowned space %v); want 2, 1, 1", vars["site_n"], vars["in_owned"], vars["in_unowned"])
	}
}

func gatheringJoinView() selectTestView {
	view := gatherNodeView(100)
	bush := view.resources[ResourceFood][0]
	farm := ResourceView{InternalID: 30, Category: ResourceFood, IsBuilding: true, Location: view.buildings[0].Location}
	for i, target := range []ResourceView{bush, bush, bush, farm} {
		view.units = append(view.units, UnitView{InternalID: uint64(10 + i), Type: UnitTypeEconomic, CurrentOrder: &CurrentOrder{Kind: OrderKindGather, TargetResource: &target}})
	}
	return view
}

const workersPerNode = `{"action":"set_var","name":"total","value":0},
	{"action":"for_each","select":{"kind":"gatherables"},"as":"node","do":[
	{"action":"set_var","name":"workers","value":{"count":{"kind":"units","where":{"in":{"order_kind":["gather"]},"if":"var(item.order_target_uid) == var(node.uid)"}}}},
	{"action":"set_var","name":"total","value":"var(total) + var(workers) * (1 + 100 * var(node.is_building))"}]}`

func TestKeyedJoinGivesTheSameAnswerAsTestingEveryItem(t *testing.T) {
	keyed := decide(t, rule(workersPerNode), gatheringJoinView(), "total")["total"]
	impure := strings.Replace(workersPerNode, `"if":"var(item.order_target_uid) == var(node.uid)"`, `"if":["var(item.order_target_uid) == var(node.uid)","random() >= 0"]`, 1)
	scanned := decide(t, rule(impure), gatheringJoinView(), "total")["total"]
	if keyed != 104 || scanned != keyed {
		t.Errorf("workers weighted by node: keyed %v, scanned %v, want 104 for both", keyed, scanned)
	}
}

func TestKeyedJoinStaysWithinTheStepBudgetOnLargeBoards(t *testing.T) {
	const n = 800
	view := selectTestView{loopTestView: loopTestView{resources: map[ResourceCategory][]ResourceView{}}, turn: 1}
	for i := 1; i <= n; i++ {
		node := ResourceView{InternalID: uint64(i), Category: ResourceFood, Location: at(i), AmountLeft: 50, GatherSpeed: 1, GatherCapacity: 5}
		view.resources[ResourceFood] = append(view.resources[ResourceFood], node)
		view.units = append(view.units, UnitView{InternalID: uint64(1000 + i), Type: UnitTypeEconomic, CurrentOrder: &CurrentOrder{Kind: OrderKindGather, TargetResource: &node}})
	}
	view.idle = view.units[:1]
	for name, config := range map[string]string{"plain": workersPerNode, "random elsewhere": workersPerNode + `,{"action":"set_var","name":"noise","value":"random()"}`} {
		if got := decide(t, rule(config), view, "total")["total"]; got != n {
			t.Errorf("%s: total workers = %v, want %d: a count per node must not rescan every unit", name, got, n)
		}
	}
}

func TestImpureSelectorStillTestsEveryItem(t *testing.T) {
	units := make([]UnitView, 10)
	for n := range units {
		units[n] = villager(uint64(n+1), Coordinate{})
	}
	view := selectTestView{loopTestView: loopTestView{units: units}, turn: 1}
	config := rule(`{"action":"set_var","name":"hits","value":{"count":{"kind":"units","where":{"if":["random() >= 0","var(item.uid) == 3"]}}}},
		{"action":"set_var","name":"next","value":"randomInt(0, 1000000)"}`)
	got := decide(t, config, view, "hits", "next")
	expected := rand.New(rand.NewSource(1))
	for range units {
		expected.Float64()
	}
	if got["hits"] != 1 || got["next"] != float64(expected.Intn(1000001)) {
		t.Errorf("hits %v, next draw %v: an equality after a random() condition must not skip the draws for the other items", got["hits"], got["next"])
	}
}

func TestLimitOneTakesTheTopRankedItemAndDrawsOnlyForTies(t *testing.T) {
	units := make([]UnitView, 6)
	for n := range units {
		units[n] = villager(uint64(n+1), Coordinate{})
	}
	view := selectTestView{loopTestView: loopTestView{units: units}, turn: 1}
	top := func(rank string) string {
		return rule(`{"action":"for_each","select":{"kind":"units","rank":"` + rank + `","limit":1},"as":"u","do":[
			{"action":"set_var","name":"picked","value":"var(u.uid)"}]},
			{"action":"set_var","name":"next","value":"randomInt(0, 1000000)"}`)
	}
	unique := decide(t, top("var(u.uid)"), view, "picked", "next")
	fresh := rand.New(rand.NewSource(1))
	if unique["picked"] != 6 || unique["next"] != float64(fresh.Intn(1000001)) {
		t.Errorf("unique max: %v, want unit 6 with no draw spent", unique)
	}
	tied := decide(t, top("0"), view, "picked", "next")
	fresh = rand.New(rand.NewSource(1))
	want := float64(fresh.Intn(6) + 1)
	if tied["picked"] != want || tied["next"] != float64(fresh.Intn(1000001)) {
		t.Errorf("six-way tie: %v, want unit %v after exactly one draw", tied, want)
	}
}

type distanceCountingView struct {
	selectTestView
	distances int
}

func (v *distanceCountingView) SpaceDistance(a Coordinate, b Coordinate) int {
	v.distances++
	return v.selectTestView.SpaceDistance(a, b)
}

func TestNearMeasuresOncePerAnchorSpaceAndHoistsTheBound(t *testing.T) {
	units := make([]UnitView, 8)
	for n := range units {
		units[n] = villager(uint64(n+1), Coordinate{X: 2 + n%2})
	}
	nearUnits := func(distance string) string {
		return rule(`{"action":"set_var","name":"found","value":{"count":{"kind":"spaces","where":{"near":{"distance":` + distance + `,"of":{"kind":"units"}}}}}}`)
	}
	view := &distanceCountingView{selectTestView: selectTestView{loopTestView: loopTestView{units: units, spaces: lineOfSpaces(make([]float64, 10)...)}, turn: 1}}
	if got := decide(t, nearUnits("1"), view, "found")["found"]; got != 4 {
		t.Fatalf("spaces within 1 of units at 2 and 3: %v, want 4", got)
	}
	baseline := &distanceCountingView{selectTestView: view.selectTestView}
	decide(t, rule(`{"action":"set_var","name":"found","value":0}`), baseline, "found")
	if lookups := view.distances - baseline.distances; lookups > 20 {
		t.Errorf("%d distance lookups, want at most one per space and anchor space (20)", lookups)
	}
	for name, distance := range map[string]string{"negative": `"0 - 1"`, "failing bound": `"1 / (turn - turn)"`} {
		if got := decide(t, nearUnits(distance), view, "found")["found"]; got != 0 {
			t.Errorf("%s bound selected %v spaces, want none", name, got)
		}
	}
	if got := decide(t, nearUnits(`"var(item.x) - 1"`), view, "found")["found"]; got != 9 {
		t.Errorf("bound reading the item: %v, want 9 (spaces 2 to 10 are within their own x - 1 of an anchor)", got)
	}
}

type revisingView struct {
	*selectTestView
	revision uint64
}

func (v *revisingView) Revision() uint64           { return v.revision }
func (v *revisingView) Issue(r IssueRequest) Order { return DispatchIssue(v, r) }
func (v *revisingView) GatherOrder(u UnitView, target ResourceView, clear bool) Order {
	for i := range v.units {
		if v.units[i].InternalID == u.InternalID {
			v.units[i].CurrentOrder = &CurrentOrder{Kind: OrderKindGather, TargetResource: &target}
		}
	}
	v.revision++
	return order(kindGather, u, 0, clear)
}

func TestKeyedJoinSeesOrdersEmittedEarlierInTheSameDecision(t *testing.T) {
	base := gatheringJoinView()
	base.units = append(base.units[:2], UnitView{InternalID: 20, Type: UnitTypeEconomic}, UnitView{InternalID: 21, Type: UnitTypeEconomic})
	view := &revisingView{selectTestView: &base}
	onBush := `{"count":{"kind":"units","where":{"in":{"order_kind":["gather"]},"if":"var(item.order_target_uid) == 2000000007"}}}`
	config := rule(`{"action":"set_var","name":"seen","value":` + onBush + `},
		{"action":"for_each","select":{"kind":"units","where":{"in":{"order_kind":["none"]},"unit_types":["economic"],"if":"var(u.uid) > 19"}},"as":"u","do":[
		{"action":"select","actors":{"kind":"units","where":{"if":"var(actor.uid) == var(u.uid)"}},"pick":{"kind":"gatherables","where":{"if":"var(target.uid) == 2000000007"}},"issue":[{"gather":{}}]},
		{"action":"set_var","name":"after","value":` + onBush + `},
		{"action":"set_var","name":"seen","value":"var(seen) * 10 + var(after)"}]}`)
	if got := decide(t, config, view, "seen")["seen"]; got != 1*100+2*10+3 {
		t.Errorf("worker counts seen = %v, want 1 then 2 then 3 as each order is emitted", got)
	}
}
