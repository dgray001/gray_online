package unit

import (
	"encoding/json"
	"io"
	"math/rand"
	"slices"
	"testing"

	. "github.com/dgray001/gray_online/game/games/risq/ai"
)

const (
	kindMoveZone uint8 = iota + 1
	kindMoveSpace
	kindAttackUnit
	kindAttackBuilding
	kindAttackSpace
	kindAttackZone
	kindBuild
	kindGather
	kindHire
	kindCancelFoundation
	kindGarrison
	kindRepair
	kindRenew
	kindBuildingAttack
)

type selectTestView struct {
	loopTestView
	idle          []UnitView
	blocked_zones []ZoneRef
	turn          int
	gold          float64

	foundations []FoundationView
}

func spaceCode(c Coordinate) int64 { return int64(c.X*100 + c.Y) }

func TestUnitActorsAreAlwaysOwnUnits(t *testing.T) {
	view := selectTestView{loopTestView: loopTestView{
		units: []UnitView{{InternalID: 1}}, enemy_units: []UnitView{{InternalID: 2, PlayerID: 1}}, spaces: lineOfSpaces(1),
	}}
	action := `{"action":"select","actors":{"kind":"units","where":{"owner":"any"}},"pick":{"kind":"spaces"},"issue":[{"move":{"to":"space"}}]}`
	orders := parseSelectModel(t, rule(action)).DecideOrders(view).Orders
	if len(orders) != 1 || !slices.Equal(orders[0].Subjects, []uint64{1}) {
		t.Errorf("orders = %+v, want one order for unit 1", orders)
	}
}

type snapshotIdleView struct {
	selectTestView
	idle_units, idle_buildings map[uint64]bool
}

func (v snapshotIdleView) IdleSnapshot() (map[uint64]bool, map[uint64]bool) {
	return v.idle_units, v.idle_buildings
}

func TestIdleFieldReadsTheViewSnapshotWhenOffered(t *testing.T) {
	view := snapshotIdleView{
		selectTestView: selectTestView{loopTestView: loopTestView{units: []UnitView{{InternalID: 1}, {InternalID: 2}}}, idle: []UnitView{{InternalID: 1}}},
		idle_units:     map[uint64]bool{2: true},
		idle_buildings: map[uint64]bool{20: true},
	}
	model := parseSelectModel(t, rule(`{"action":"set_var","name":"idle_unit","value":{"sum":{"kind":"units","where":{"if":"var(item.idle)"},"of":"var(item.uid)"}}},
		{"action":"set_var","name":"idle_buildings","value":{"count":{"kind":"buildings","where":{"if":"var(item.idle)"}}}}`))
	model.DecideOrders(view)
	if unit, _ := model.Var("idle_unit"); unit != 2 {
		t.Errorf("idle unit uid = %v, want the snapshot's unit 2 and not IdleUnits' unit 1", unit)
	}
	if buildings, _ := model.Var("idle_buildings"); buildings != 1 {
		t.Errorf("idle buildings = %v, want the snapshot's one building", buildings)
	}
}

func order(kind uint8, u UnitView, target int64, clear bool) Order {
	return Order{Subjects: []uint64{u.InternalID}, OrderType: kind, TargetID: target, ClearPreviousOrders: clear}
}

func (v selectTestView) Issue(r IssueRequest) Order { return DispatchIssue(v, r) }
func (v selectTestView) Population() (int, int)     { return 0, 100 }
func (v selectTestView) HireMercenaryOrder(unit_id uint32, t ZoneRef) Order {
	return Order{OrderType: kindHire, TargetID: spaceCode(t.Space)}
}
func (v selectTestView) AllBuildings() []BuildingView {
	return slices.Concat(v.Buildings(), v.enemy_buildings)
}
func (v selectTestView) Buildings() []BuildingView {
	return append(slices.Clone(v.buildings), BuildingView{InternalID: 20, BuildingID: 1})
}
func (v selectTestView) IdleUnits() []UnitView         { return v.idle }
func (v selectTestView) IdleBuildings() []BuildingView { return nil }
func (v selectTestView) Foundations() []FoundationView { return v.foundations }
func (v selectTestView) CancelFoundationOrder(f FoundationView) Order {
	return Order{OrderType: kindCancelFoundation, TargetID: spaceCode(f.Location.Space)}
}
func (v selectTestView) TurnNumber() int               { return v.turn }
func (v selectTestView) BuildingAvailable(uint32) bool { return true }
func (v selectTestView) ValidBuildZone(_ uint32, target ZoneRef) bool {
	return !slices.Contains(v.blocked_zones, target)
}
func (v selectTestView) BuildCost(uint32) Cost { return Cost{Wood: 10} }
func (v selectTestView) Resource(c ResourceCategory) float64 {
	if c == ResourceGold && v.gold > 0 {
		return v.gold
	}
	return 100
}
func (v selectTestView) LocationDistance(a ZoneRef, b ZoneRef) int {
	if a.Space == b.Space {
		dx, dy := b.Zone.X-a.Zone.X, b.Zone.Y-a.Zone.Y
		return max(max(dx, -dx), max(dy, -dy), max(dx+dy, -dx-dy))
	}
	return 6 * v.SpaceDistance(a.Space, b.Space)
}
func (v selectTestView) SpaceNeighbors(space Coordinate) []Coordinate {
	return nil
}

func (v selectTestView) GarrisonOrder(u UnitView, t BuildingView, clear bool) Order {
	return order(kindGarrison, u, int64(t.InternalID), clear)
}
func (v selectTestView) RepairOrder(u UnitView, t BuildingView, clear bool) Order {
	return order(kindRepair, u, int64(t.InternalID), clear)
}
func (v selectTestView) RenewOrder(u UnitView, t BuildingView, clear bool) Order {
	return order(kindRenew, u, int64(t.InternalID), clear)
}
func (v selectTestView) BuildingAttackUnitOrder(b BuildingView, t UnitView) Order {
	return Order{Subjects: []uint64{b.InternalID}, OrderType: kindBuildingAttack, TargetID: int64(t.InternalID)}
}
func (v selectTestView) MoveOrder(u UnitView, t ZoneRef, clear bool) Order {
	return order(kindMoveZone, u, spaceCode(t.Space), clear)
}
func (v selectTestView) MoveSpaceOrder(u UnitView, t Coordinate, clear bool) Order {
	return order(kindMoveSpace, u, spaceCode(t), clear)
}
func (v selectTestView) AttackUnitOrder(u UnitView, t UnitView, clear bool) Order {
	return order(kindAttackUnit, u, int64(t.InternalID), clear)
}
func (v selectTestView) AttackBuildingOrder(u UnitView, t BuildingView, clear bool) Order {
	return order(kindAttackBuilding, u, int64(t.InternalID), clear)
}
func (v selectTestView) AttackSpaceOrder(u UnitView, t Coordinate, clear bool) Order {
	return order(kindAttackSpace, u, spaceCode(t), clear)
}
func (v selectTestView) AttackZoneOrder(u UnitView, t ZoneRef, clear bool) Order {
	return order(kindAttackZone, u, spaceCode(t.Space), clear)
}
func (v selectTestView) BuildOrder(u UnitView, _ uint32, t ZoneRef, clear bool) Order {
	return order(kindBuild, u, spaceCode(t.Space), clear)
}
func (v selectTestView) GatherOrder(u UnitView, t ResourceView, clear bool) Order {
	return order(kindGather, u, int64(t.InternalID), clear)
}

func parseSelectModel(t *testing.T, config string) *RulesModel {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal([]byte(config), &raw); err != nil {
		t.Fatal(err)
	}
	model, ok := ParseModelWithDiagnostics(raw, rand.New(rand.NewSource(1)), io.Discard).(*RulesModel)
	if !ok {
		t.Fatalf("config did not load: %s", config)
	}
	return model
}

func selectOrders(t *testing.T, config string, view View) []Order {
	t.Helper()
	return parseSelectModel(t, config).DecideOrders(view).Orders
}

func villager(id uint64, space Coordinate) UnitView {
	return UnitView{InternalID: id, Type: UnitTypeEconomic, Location: ZoneRef{Space: space}, Builds: []Producible{{ID: 2}}}
}

func TestSelectMovesVillagerToTopRankedEmptySpaceAndBuilds(t *testing.T) {
	worker := villager(1, Coordinate{})
	view := selectTestView{
		loopTestView: loopTestView{
			units:     []UnitView{worker},
			resources: map[ResourceCategory][]ResourceView{ResourceFood: {{InternalID: 5, Location: ZoneRef{Space: Coordinate{X: 2}}}}},
			spaces: []SpaceInfo{
				{Space: Coordinate{}, GoldIncome: 50},
				{Space: Coordinate{X: 3}, GoldIncome: 2},
				{Space: Coordinate{X: 2, Y: 1}, GoldIncome: 5},
			},
			zones: []ZoneInfo{
				{Location: ZoneRef{Space: Coordinate{}}},
				{Location: ZoneRef{Space: Coordinate{X: 3}}},
				{Location: ZoneRef{Space: Coordinate{X: 2, Y: 1}}},
			},
		},
		idle: []UnitView{worker},
		turn: 1,
	}
	config := rule(`{"action":"select",
		"pick":{"kind":"zones","where":{"near":{"distance":1,"of":{"kind":"resources"}}},
		        "rank":"var(target.space.gold_income)","limit":1},
		"actors":{"kind":"units","where":{"if":"var(actor.idle)"}},
		"issue":[{"move":{}},{"build":{"building_id":2}}]}`)
	got := selectOrders(t, config, view)
	want := []Order{order(kindMoveZone, worker, 201, true), order(kindBuild, worker, 201, false)}
	if !slices.EqualFunc(got, want, func(a Order, b Order) bool {
		return a.OrderType == b.OrderType && a.TargetID == b.TargetID && a.ClearPreviousOrders == b.ClearPreviousOrders && slices.Equal(a.Subjects, b.Subjects)
	}) {
		t.Fatalf("orders = %+v, want %+v", got, want)
	}
}

func TestSelectIntoSetThenActOnIt(t *testing.T) {
	units := []UnitView{
		{InternalID: 1, Health: 4, CanAttack: true}, {InternalID: 2, Health: 6, CanAttack: true}, {InternalID: 3, Health: 9, CanAttack: true},
	}
	view := selectTestView{
		loopTestView: loopTestView{units: units, enemy_buildings: []BuildingView{{InternalID: 21, PlayerID: 1, Location: ZoneRef{Space: Coordinate{X: 3}}}}},
		idle:         units,
		turn:         1,
	}
	config := `{"rules": [{"when": {"always": {}}, "then": [
		{"action":"select","pick":{"kind":"units","where":{"if":"var(target.health) - 5"},"rank":"var(target.health)","limit":2},"into":{"set":"strong","var":"strong_n"}},
		{"action":"select","pick":{"kind":"buildings","where":{"owner":"enemy"}},
		 "actors":{"kind":"units","where":{"in_set":"strong"}},"issue":[{"attack":{}}]}]}]}`
	var subjects []uint64
	for _, o := range selectOrders(t, config, view) {
		if o.OrderType != kindAttackBuilding || o.TargetID != 21 {
			t.Fatalf("unexpected order %+v", o)
		}
		subjects = append(subjects, o.Subjects...)
	}
	if !slices.Equal(subjects, []uint64{2, 3}) {
		t.Fatalf("attackers = %v, want [2 3]", subjects)
	}
}

func TestSelectPersistentSetOutlivesTheTurn(t *testing.T) {
	units := []UnitView{{InternalID: 1, CanAttack: true}, {InternalID: 2, CanAttack: true}}
	view := selectTestView{loopTestView: loopTestView{units: units, enemy_buildings: []BuildingView{{InternalID: 21, PlayerID: 1}}}, idle: units, turn: 1}
	config := `{"rules": [
		{"when": {"value_equals": {"value": "gameVar(turn)", "amount": 1}}, "then": [{"action":"select","pick":{"kind":"units","where":{"if":"var(target.uid) - 1"}},"into":{"set":"kept","persist":true}}]},
		{"when": {"always": {}}, "then": [{"action":"select","pick":{"kind":"buildings","where":{"owner":"enemy"}},"actors":{"kind":"units","where":{"in_set":"kept"}},"issue":[{"attack":{}}]}]}]}`
	model := parseSelectModel(t, config)
	model.DecideOrders(view)
	view.turn = 2
	orders := model.DecideOrders(view).Orders
	if len(orders) != 1 || orders[0].Subjects[0] != 2 {
		t.Fatalf("turn 2 orders = %+v, want unit 2 attacking from the persisted set", orders)
	}
}

func TestSelectIntoBucketReplacesMembers(t *testing.T) {
	units := []UnitView{{InternalID: 1, Health: 1, CanAttack: true}, {InternalID: 2, Health: 2, CanAttack: true}, {InternalID: 3, Health: 3, CanAttack: true}}
	view := selectTestView{loopTestView: loopTestView{units: units, enemy_buildings: []BuildingView{{InternalID: 21, PlayerID: 1}}}, idle: units, turn: 1}
	config := rule(`{"action":"select","pick":{"kind":"units","rank":"var(target.health)","limit":2},"into":{"bucket":"scouts"}},
		{"action":"select","pick":{"kind":"buildings","where":{"owner":"enemy"}},"actors":{"kind":"units","where":{"bucket":"scouts"}},"issue":[{"attack":{}}]}`)
	orders := selectOrders(t, config, view)
	got := []uint64{}
	for _, o := range orders {
		got = append(got, o.Subjects...)
	}
	if !slices.Equal(got, []uint64{2, 3}) {
		t.Fatalf("bucket attackers = %v, want the two healthiest units [2 3]", got)
	}
}

func TestSelectBucketMembershipModes(t *testing.T) {
	units := []UnitView{{InternalID: 1}, {InternalID: 2}, {InternalID: 3}}
	view := selectTestView{loopTestView: loopTestView{units: units}, idle: units}
	for mode, want := range map[string][]uint64{"": {2, 3}, `,"mode":"replace"`: {2, 3}, `,"mode":"add"`: {1, 2, 3}, `,"mode":"remove"`: {1}} {
		config := rule(`{"action":"select","pick":{"kind":"units","where":{"in":{"uid":[1,2]}}},"into":{"bucket":"group"}},
			{"action":"select","pick":{"kind":"units","where":{"in":{"uid":[2,3]}}},"into":{"bucket":"group"` + mode + `}},
			{"action":"select","pick":{"kind":"units"},"actors":{"kind":"units","where":{"bucket":"group"}},"issue":[{"move":{}}]}`)
		var got []uint64
		for _, order := range selectOrders(t, config, view) {
			got = append(got, order.Subjects...)
		}
		if !slices.Equal(got, want) {
			t.Errorf("mode %q: got %v, want %v", mode, got, want)
		}
	}
}

func TestSelectAssignSpreadsActorsWithinCapacity(t *testing.T) {
	units := []UnitView{{InternalID: 1, CanAttack: true}, {InternalID: 2, CanAttack: true}, {InternalID: 3, CanAttack: true}, {InternalID: 4, CanAttack: true}}
	view := selectTestView{
		loopTestView: loopTestView{units: units, enemy_buildings: []BuildingView{
			{InternalID: 21, PlayerID: 1, Location: ZoneRef{Space: Coordinate{X: 1}}},
			{InternalID: 22, PlayerID: 1, Location: ZoneRef{Space: Coordinate{X: 2}}},
		}},
		idle: units, turn: 1,
	}
	config := rule(`{"action":"select","pick":{"kind":"buildings","where":{"owner":"enemy"},"capacity":2,"rank":"100 - var(target.from_base)",
		"let":{"from_base":{"min":{"kind":"buildings","where":{"owner":"own","in":{"building_id":[1]}},"of":"var(item.distance_to_target)"}}}},
		"actors":{"kind":"units"},"issue":[{"attack":{}}]}`)
	counts := map[int64]int{}
	for _, o := range selectOrders(t, config, view) {
		counts[o.TargetID]++
	}
	if counts[21] != 2 || counts[22] != 2 {
		t.Fatalf("attackers per building = %v, want 2 each", counts)
	}
}

func TestSelectSpaceAndZoneVerbs(t *testing.T) {
	worker := villager(1, Coordinate{})
	worker.CanAttack = true
	view := selectTestView{
		loopTestView: loopTestView{
			units:  []UnitView{worker},
			spaces: []SpaceInfo{{Space: Coordinate{X: 1}}},
			zones:  []ZoneInfo{{Location: ZoneRef{Space: Coordinate{X: 1}}, BuildingPlayer: -1}},
		},
		idle: []UnitView{worker}, turn: 1,
	}
	cases := map[string]struct {
		kind, verb string
		want       uint8
	}{
		"move space":   {"spaces", "move", kindMoveSpace},
		"move zone":    {"zones", "move", kindMoveZone},
		"attack space": {"spaces", "attack", kindAttackSpace},
		"attack zone":  {"zones", "attack", kindAttackZone},
	}
	for name, c := range cases {
		config := rule(`{"action":"select","pick":{"kind":"` + c.kind + `"},"issue":[{"` + c.verb + `":{}}]}`)
		orders := selectOrders(t, config, view)
		if len(orders) != 1 || orders[0].OrderType != c.want {
			t.Errorf("%s: orders = %+v, want one of type %d", name, orders, c.want)
		}
	}
}

func hireView() selectTestView {
	return selectTestView{
		loopTestView: loopTestView{
			mercenaries: []Producible{{ID: 101, Kind: ProducibleUnit, Cost: Cost{Gold: 6}, StaminaCost: 2}},
			zones:       []ZoneInfo{{Location: at(1)}, {Location: at(2)}, {Location: at(3)}, {Location: at(4)}},
			spaces: []SpaceInfo{
				{Space: Coordinate{X: 1}, GoldIncome: 4}, {Space: Coordinate{X: 2}, GoldIncome: 9},
				{Space: Coordinate{X: 3}, GoldIncome: 7}, {Space: Coordinate{X: 4}, GoldIncome: 1},
			},
		},
		turn: 1,
	}
}

func hireCounts(orders []Order) map[int64]int {
	counts := map[int64]int{}
	for _, o := range orders {
		if o.OrderType != kindHire {
			panic("not a hire order")
		}
		counts[o.TargetID]++
	}
	return counts
}

func TestSelectHireCountAppliesToEachRankedZone(t *testing.T) {
	config := rule(`{"action":"select","pick":{"kind":"zones","rank":"var(target.space.gold_income)","limit":2},
		"issue":[{"hire":{"unit_id":101,"count":3}}]}`)
	counts := hireCounts(selectOrders(t, config, hireView()))
	if counts[200] != 3 || counts[300] != 3 || len(counts) != 2 {
		t.Fatalf("hires per zone = %v, want 3 each at the two richest zones", counts)
	}
}

func TestSelectHireAtThirdRankedZone(t *testing.T) {
	config := rule(`{"action":"select","pick":{"kind":"zones","rank":"var(target.space.gold_income)","skip":2,"limit":1},
		"issue":[{"hire":{"unit_id":101}}]}`)
	counts := hireCounts(selectOrders(t, config, hireView()))
	if len(counts) != 1 || counts[100] != 1 {
		t.Fatalf("hires = %v, want one hire at the third richest space (x=1)", counts)
	}
}

func TestSelectHireCountControlsSpending(t *testing.T) {
	poor := hireView()
	poor.gold = 13
	hired := func(count string) int {
		config := rule(`{"action":"select","pick":{"kind":"zones","limit":1},"issue":[{"hire":{"unit_id":101,"count":` + count + `}}]}`)
		total := 0
		for _, n := range hireCounts(selectOrders(t, config, poor)) {
			total += n
		}
		return total
	}
	if got := hired("10"); got != 10 {
		t.Fatalf("hired %d with 13 gold at 6 each, want all 10 since overspending is the rule's call", got)
	}
	if got := hired(`"min(10, round(gameVar(gold) / 6 - 0.5))"`); got != 2 {
		t.Fatalf("hired %d with a count bounded by gold, want 2", got)
	}
	unknown := rule(`{"action":"select","pick":{"kind":"zones"},"issue":[{"hire":{"unit_id":999}}]}`)
	if got := selectOrders(t, unknown, hireView()); len(got) != 0 {
		t.Fatalf("hired a mercenary that is not available: %+v", got)
	}
}

func TestHireCountCanUseWhatIsStillAffordable(t *testing.T) {
	view := hireView()
	view.gold = 100
	hire := func(count string) string {
		return `{"action":"select","pick":{"kind":"zones","limit":1},"issue":[{"hire":{"unit_id":101,"count":` + count + `}}]}`
	}
	config := `{"rules":[{"when":{"always":{}},"then":[` + hire("8") + `,` + hire(`"0.5 * var(item.can_afford)"`) + `]}]}`
	total := 0
	for _, n := range hireCounts(selectOrders(t, config, view)) {
		total += n
	}
	if total != 12 {
		t.Fatalf("hired %d, want 12: 8 first (48 of 100 gold), then half of the 8 still affordable with 52 gold left", total)
	}
}

func TestIssueDispatchesEveryVerb(t *testing.T) {
	unit, building := UnitView{InternalID: 1}, BuildingView{InternalID: 2}
	target := IssueTarget{Unit: &UnitView{InternalID: 3}, Building: &BuildingView{InternalID: 4}, Resource: &ResourceView{InternalID: 5}}
	want := map[Verb]uint8{
		VerbMoveSpace: 21, VerbMoveZone: 1, VerbGather: 2, VerbBuild: 3, VerbRepair: 4, VerbRenew: 5, VerbAttackUnit: 6,
		VerbAttackBuilding: 7, VerbAttackSpace: 8, VerbAttackZone: 9, VerbGarrison: 10, VerbUngarrison: 11, VerbDeleteUnit: 12,
		VerbCreateUnit: 13, VerbResearch: 14, VerbDeleteBuilding: 15, VerbBuildingAttackUnit: 16, VerbBuildingAttackBuilding: 17,
		VerbCancelFoundation: 18, VerbHire: 19, VerbCancelOrder: 20,
	}
	for verb, kind := range want {
		view := &fakeView{}
		view.Issue(IssueRequest{Verb: verb, Unit: unit, Building: building, Target: target})
		if len(view.calls) != 1 || view.calls[0].Kind != kind {
			t.Errorf("verb %d recorded %+v, want one call of kind %d", verb, view.calls, kind)
		}
	}
	view := &fakeView{}
	view.Issue(IssueRequest{Verb: VerbAttackBuilding, Unit: unit, Target: target, Clear: true})
	if call := view.calls[0]; call.Subject != 1 || call.Target != 4 || !call.Clear {
		t.Errorf("attack building call %+v, want subject 1 target 4 clear", call)
	}
}

func TestSelectConfigErrors(t *testing.T) {
	cases := map[string]string{
		"unknown kind":           `{"action":"select","pick":{"kind":"ships"},"into":{"set":"x"}}`,
		"unknown verb":           `{"action":"select","pick":{"kind":"spaces"},"issue":[{"teleport":{}}]}`,
		"missing pick":           `{"action":"select","issue":[{"attack":{}}]}`,
		"unknown rank field":     `{"action":"select","pick":{"kind":"spaces","rank":"var(target.nope)"},"into":{"set":"x"}}`,
		"unknown actor field":    `{"action":"select","pick":{"kind":"spaces"},"actors":{"kind":"units","where":{"if":"var(actor.nope)"}},"issue":[{"move":{}}]}`,
		"bucket of buildings":    `{"action":"select","pick":{"kind":"buildings"},"into":{"bucket":"b"}}`,
		"set and bucket":         `{"action":"select","pick":{"kind":"units"},"into":{"set":"s","bucket":"b"}}`,
		"actors without issue":   `{"action":"select","pick":{"kind":"units"},"actors":{"kind":"units"}}`,
		"build with zero id":     `{"action":"select","pick":{"kind":"spaces"},"issue":[{"build":{"building_id":0}}]}`,
		"cancel with actors":     `{"action":"select","pick":{"kind":"foundations"},"actors":{"kind":"units"},"issue":[{"cancel_foundation":{}}]}`,
		"cancel with assign":     `{"action":"select","pick":{"kind":"foundations"},"assign":{"capacity":1},"issue":[{"cancel_foundation":{}}]}`,
		"unknown key":            `{"action":"select","pick":{"kind":"spaces"},"issue":[{"move":{}}],"orders":["move"]}`,
		"hire with actors":       `{"action":"select","pick":{"kind":"zones"},"actors":{"kind":"units"},"issue":[{"hire":{"unit_id":101}}]}`,
		"hire with assign":       `{"action":"select","pick":{"kind":"zones"},"assign":{"capacity":2},"issue":[{"hire":{"unit_id":101}}]}`,
		"hire with space pick":   `{"action":"select","pick":{"kind":"spaces"},"issue":[{"hire":{"unit_id":101}}]}`,
		"hire beside a verb":     `{"action":"select","pick":{"kind":"zones"},"issue":[{"hire":{"unit_id":101}},{"move":{}}]}`,
		"hire without unit id":   `{"action":"select","pick":{"kind":"zones"},"issue":[{"hire":{}}]}`,
		"bucket filter on zone":  `{"action":"select","pick":{"kind":"zones","where":{"bucket":"b"}},"into":{"set":"x"}}`,
		"delete without actors":  `{"action":"select","pick":{"kind":"spaces"},"issue":[{"delete":{}}]}`,
		"through with unit zone": `{"action":"select","pick":{"kind":"spaces","where":{"near":{"distance":2,"of":{"kind":"spaces"},"unit":"zone","through":{"if":"1"}}}},"into":{"set":"x"}}`,
		"through not an object":  `{"action":"select","pick":{"kind":"spaces","where":{"near":{"distance":2,"of":{"kind":"spaces"},"through":"invalid"}}},"into":{"set":"x"}}`,
		"through unknown field":  `{"action":"select","pick":{"kind":"spaces","where":{"near":{"distance":2,"of":{"kind":"spaces"},"through":{"if":"var(actor.idle)"}}}},"into":{"set":"x"}}`,
	}
	for name, action := range cases {
		if loadError(rule(action)) == nil {
			t.Errorf("%s: malformed select accepted", name)
		}
	}
	explicit := rule(`{"action":"select","actors":{"kind":"units","where":{"if":"var(actor.idle)"}},"pick":{"kind":"spaces"},"issue":[{"delete":{}}]}`)
	if err := loadError(explicit); err != nil {
		t.Errorf("delete with explicit actors rejected: %v", err)
	}
}
