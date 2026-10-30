package ai

import (
	"encoding/json"
	"io"
	"math/rand"

	"github.com/dgray001/gray_online/util"
)

// Set once at startup, before any game runs; per-rule step counts are only written to debug logs
var LogRuleSteps bool

type Bucket struct {
	Members map[uint64]bool
}

type Internals struct {
	diagnostics        io.Writer
	rng                *rand.Rand
	Buckets            map[string]*Bucket
	bucket_members     map[uint64]bool
	pending            Cost
	pending_population int
	renewals           map[uint64]bool
	behaviors          []UnitBehavior
	building_behaviors []BuildingBehavior
	rule_index         int
	// set_var values: turn_vars are cleared every turn, vars persist for the whole game
	turn_vars    map[string]float64
	vars         map[string]float64
	warned       map[string]bool
	assign_round float64
	// the item each enclosing for_each is visiting by its "as" name, and the loop bodies run so far this decision
	loop_items         map[string]loopItem
	loop_steps         int
	select_census      *census
	select_steps       int
	eval_errors        uint64
	binding_depth      int
	turn_sets          map[string]*entitySet
	joins              map[*selectorSpec]*joinIndex
	world              *worldIndex
	sets               map[string]*entitySet
	target_queue_index map[uint64]map[uint64]bool
	last_observed      ObservationHistory
	assignments        map[string]*assignJob
}

func (i *Internals) Refresh() {
	i.pending = Cost{}
	i.pending_population = 0
	i.renewals = make(map[uint64]bool)
	i.behaviors = nil
	i.building_behaviors = nil
	i.turn_vars = nil
	i.turn_sets = nil
	i.assignments = nil
	i.joins = nil
	i.world = nil
	i.loop_steps = 0
	i.select_steps = 0
	i.eval_errors = 0
}

func (i *Internals) spend(cost Cost) {
	i.pending = Cost{Food: i.pending.Food + cost.Food, Wood: i.pending.Wood + cost.Wood, Stone: i.pending.Stone + cost.Stone, Gold: i.pending.Gold + cost.Gold}
}

func (i *Internals) available(view View, category ResourceCategory) float64 {
	return view.Resource(category) - i.pending.of(category)
}

func (i *Internals) population(view View) (int, int) {
	current, limit := view.Population()
	return current + i.pending_population, limit
}

func (i *Internals) bucket(name string) *Bucket {
	if i.Buckets == nil {
		i.Buckets = make(map[string]*Bucket)
	}
	b, ok := i.Buckets[name]
	if !ok {
		b = &Bucket{Members: make(map[uint64]bool)}
		i.Buckets[name] = b
	}
	return b
}

func (i *Internals) pruneBuckets(view View) {
	i.bucket_members = nil
	if len(i.Buckets) == 0 {
		return
	}
	alive := make(map[uint64]bool)
	for _, u := range view.Units() {
		alive[u.InternalID] = true
	}
	for _, b := range i.Buckets {
		for id := range b.Members {
			if !alive[id] {
				delete(b.Members, id)
			}
		}
	}
}

func (i *Internals) bucketMembers() map[uint64]bool {
	if i.bucket_members == nil {
		i.bucket_members = make(map[uint64]bool)
		for _, bucket := range i.Buckets {
			for id := range bucket.Members {
				i.bucket_members[id] = true
			}
		}
	}
	return i.bucket_members
}

type Rule struct {
	when Condition
	then []Action
}

type RulesModel struct {
	rules     []Rule
	internals Internals
}

func (m *RulesModel) ApplyUpdate(view View, kind string) {
	if kind == "start-turn" {
		if m.internals.last_observed == nil {
			m.internals.last_observed = make(ObservationHistory)
		}
		m.internals.last_observed.Observe(view.AllSpaces(), view.TurnNumber())
	}
}

func logTurnData(view View) {
	if !util.IsDebug(view.Logger()) {
		return
	}
	current, limit := view.Population()
	data, _ := json.Marshal(map[string]interface{}{
		"population": map[string]int{"current": current, "limit": limit},
		"resources": map[string]float64{
			"food":  view.Resource(ResourceFood),
			"wood":  view.Resource(ResourceWood),
			"stone": view.Resource(ResourceStone),
			"gold":  view.Resource(ResourceGold),
		},
		"units":           view.Units(),
		"buildings":       view.Buildings(),
		"foundations":     view.Foundations(),
		"enemy_units":     view.VisibleEnemyUnits(),
		"enemy_buildings": view.VisibleEnemyBuildings(),
		"spaces":          view.AllSpaces(),
	})
	util.Debugf(view.Logger(), "ai %s: new turn data %s", view.Nickname(), data)
}

// Debug log only: every script variable's value at the end of the turn, so a replay shows why rules fired
func logVars(view View, internals *Internals) {
	if !util.IsDebug(view.Logger()) {
		return
	}
	vars := make(map[string]float64, len(internals.vars)+len(internals.turn_vars))
	for k, v := range internals.vars {
		vars[k] = v
	}
	for k, v := range internals.turn_vars {
		vars[k] = v
	}
	data, _ := json.Marshal(vars)
	util.Debugf(view.Logger(), "ai %s: vars %s", view.Nickname(), data)
}

// Lets tests read what the last decision left in a script variable
func (m *RulesModel) Var(name string) (float64, bool) {
	if v, ok := m.internals.turn_vars[name]; ok {
		return v, true
	}
	v, ok := m.internals.vars[name]
	return v, ok
}

func (m *RulesModel) DecideOrders(view View) Decision {
	logTurnData(view)
	m.internals.Refresh()
	m.internals.pruneBuckets(view)
	orders := make([]Order, 0)
	for rule_index, rule := range m.rules {
		m.internals.rule_index, m.internals.select_steps, m.internals.loop_steps = rule_index, 0, 0
		matches, valid := evaluateCondition(rule.when, view, &m.internals)
		if valid && matches {
			for _, action := range rule.then {
				orders = append(orders, action.ToOrders(view, &m.internals)...)
			}
		}
		if LogRuleSteps {
			util.Debugf(view.Logger(), "ai %s: rule %d steps select=%d loop=%d", view.Nickname(), rule_index, m.internals.select_steps, m.internals.loop_steps)
		}
	}
	util.Debugf(view.Logger(), "ai %s: submitting orders %+v", view.Nickname(), orders)
	logVars(view, &m.internals)
	return Decision{Orders: orders, Behaviors: m.internals.behaviors, BuildingBehaviors: m.internals.building_behaviors}
}
