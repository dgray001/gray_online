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

func (r *GameRisq) stateHash(dump io.Writer) uint64 {
	hasher := fnv.New64a()
	h := io.MultiWriter(hasher, dump)
	for _, id := range slices.Sorted(maps.Keys(r.units)) {
		u := r.units[id]
		garrison := uint64(0)
		if u.garrisoned_in != nil {
			garrison = u.garrisoned_in.internal_id
		}
		fmt.Fprintf(h, "u%d p%d t%d z%s h%s s%d g%d d%t;", id, u.player_id, u.unit_id, hashZone(u.zone), hashFloat(u.cs.health), u.current_stamina, garrison, u.deleted)
		for _, o := range u.order_queue.active_orders {
			fmt.Fprintf(h, "o%d:%d:%t:%t;", o.order_type, o.target_id, o.executed, o.cancelled)
		}
	}
	for _, id := range slices.Sorted(maps.Keys(r.buildings)) {
		b := r.buildings[id]
		queue := make([]string, 0, len(b.production_queue))
		for _, item := range b.production_queue {
			queue = append(queue, fmt.Sprintf("%d:%d:%d", item.kind, item.item_id, item.stamina_remaining))
		}
		slices.Sort(queue)
		fmt.Fprintf(h, "b%d p%d t%d z%s h%s c%d r%s d%t q%v;", id, b.player_id, b.building_id, hashZone(b.zone), hashFloat(b.cs.health), b.stamina_remaining, hashFloat(b.resources_left), b.deleted, queue)
		for _, o := range b.order_queue.active_orders {
			fmt.Fprintf(h, "o%d:%d:%t:%t;", o.order_type, o.target_id, o.executed, o.cancelled)
		}
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
		fmt.Fprint(h, ";")
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
