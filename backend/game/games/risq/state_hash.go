package risq

import (
	"fmt"
	"hash/fnv"
	"io"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/dgray001/gray_online/util"
)

// Logs a hash of all engine-truth state for determinism debugging; no-op unless DebugLog is enabled.
func (r *GameRisq) logStateHash(tick_label string) {
	if util.DebugLog.Writer() == io.Discard {
		return
	}
	var dump strings.Builder
	util.DebugLog.Printf("state_hash turn=%d tick=%s hash=%016x", r.turn_number, tick_label, r.stateHash(&dump))
	if os.Getenv("RISQ_STATE_DUMP") == "1" {
		util.DebugLog.Printf("state_dump turn=%d tick=%s %s", r.turn_number, tick_label, dump.String())
	}
}

func hashFloat(x float64) string {
	return strconv.FormatFloat(util.RoundTo(x, gatherRoundingPlaces)+0, 'f', gatherRoundingPlaces, 64)
}

func hashZone(z *RisqZone) string {
	if z == nil {
		return "-"
	}
	return fmt.Sprintf("%d/%d", z.space.coordinate_key, z.coordinate_key)
}

func hashBehavior(h io.Writer, o *orderableBase) {
	fmt.Fprintf(h, " ic%t tp%v ts%d", o.interrupt_current, o.target_priority, o.turn_stamina)
	attackers := make([]string, 0, len(o.attacked_by))
	for _, event := range o.attacked_by {
		attackers = append(attackers, fmt.Sprintf("%d:%d:%d", event.tick, event.attacker_type, event.attacker_id))
	}
	slices.Sort(attackers)
	fmt.Fprintf(h, " ab%v", attackers)
}

// Order internal ids are left out: they depend on submission arrival order, not on game state
func hashOrders(h io.Writer, orders []*RisqOrder) {
	for _, o := range orders {
		fmt.Fprintf(h, "o%d:%d:%t:%t:%t:%t:%d:%d:%v;", o.order_type, o.target_id, o.executed, o.cancelled, o.clear_previous_orders, o.received, o.turn_received, o.turn_resolved, slices.Sorted(maps.Keys(o.subjects)))
	}
}

func hashGatherPoint(h io.Writer, gp *RisqGatherPoint) {
	if gp != nil {
		fmt.Fprintf(h, " gp%d:%d:%d:%d", gp.location_kind, gp.location_id, gp.object_type, gp.object_id)
	}
}

func (r *GameRisq) stateHash(dump io.Writer) uint64 {
	hasher := fnv.New64a()
	h := io.MultiWriter(hasher, dump)
	for _, id := range slices.Sorted(maps.Keys(r.units)) {
		u := r.units[id]
		garrison := uint64(0)
		if u.garrisoned_in != nil {
			garrison = u.garrisoned_in.internal_id
		}
		fmt.Fprintf(h, "u%d p%d t%d z%s h%s s%d g%d d%t sn%d bk%t", id, u.player_id, u.unit_id, hashZone(u.zone), hashFloat(u.cs.health), u.current_stamina, garrison, u.deleted, u.stance, u.attack_back)
		hashBehavior(h, &u.orderableBase)
		fmt.Fprint(h, ";")
		hashOrders(h, u.order_queue.active_orders)
	}
	for _, id := range slices.Sorted(maps.Keys(r.buildings)) {
		b := r.buildings[id]
		queue := make([]string, 0, len(b.production_queue))
		for _, item := range b.production_queue {
			queue = append(queue, fmt.Sprintf("%d:%d:%d", item.kind, item.item_id, item.stamina_remaining))
		}
		slices.Sort(queue)
		fmt.Fprintf(h, "b%d p%d t%d z%s h%s c%d r%s d%t q%v", id, b.player_id, b.building_id, hashZone(b.zone), hashFloat(b.cs.health), b.stamina_remaining, hashFloat(b.resources_left), b.deleted, queue)
		fmt.Fprintf(h, " cs%d hs%d ct%d aa%t gu%v rn%t pr%s", b.current_stamina, b.health_synced_stamina, b.construction_stamina_total, b.auto_attack, slices.Sorted(maps.Keys(b.garrisoned_units)), b.renewing != nil, hashFloat(b.pending_renew))
		hashBehavior(h, &b.orderableBase)
		hashGatherPoint(h, b.gather_point)
		fmt.Fprint(h, ";")
		hashOrders(h, b.order_queue.active_orders)
	}
	r.hashPlayersAndBoard(h)
	return hasher.Sum64()
}

func (r *GameRisq) hashPlayersAndBoard(h io.Writer) {
	for _, p := range r.players {
		res := p.resources
		fmt.Fprintf(h, "p%d f%s w%s s%s g%s pop%d/%d", p.player.Player_id, hashFloat(res.food), hashFloat(res.wood), hashFloat(res.stone), hashFloat(res.gold), nonDeletedUnitCount(p.units), p.populationLimit())
		for _, tech_id := range slices.Sorted(maps.Keys(p.researched_techs)) {
			fmt.Fprintf(h, " t%d:%t", tech_id, p.researched_techs[tech_id])
		}
		for _, key := range slices.Sorted(maps.Keys(p.planned_foundations)) {
			fmt.Fprintf(h, " pf%d:%d", key, p.planned_foundations[key].building_id)
		}
		fmt.Fprintf(h, " el%t am%v;", p.eliminated, slices.Sorted(maps.Keys(p.available_mercenaries)))
	}
	for _, space := range r.allSpaces() {
		fmt.Fprintf(h, "s%d o%d t%d", space.coordinate_key, space.ownership, space.terrain_id)
		for _, p := range r.players {
			fmt.Fprintf(h, " v%d", space.getVisibility(p.player.Player_id))
		}
		for _, row := range space.zones {
			for _, zone := range row {
				fmt.Fprintf(h, " z%d o%d t%d", zone.coordinate_key, zone.ownership, zone.terrain_override)
				if zone.resource != nil {
					fmt.Fprintf(h, " r%d:%d:%s", zone.resource.internal_id, zone.resource.resource_id, hashFloat(zone.resource.resources_left))
				}
				if zone.building != nil {
					fmt.Fprintf(h, " b%d", zone.building.internal_id)
				}
			}
		}
		fmt.Fprint(h, ";")
	}
}
