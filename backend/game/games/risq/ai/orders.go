package ai

import "fmt"

type orderItem struct{ entry OrderEntry }

func (o orderItem) field(name string) (float64, bool) { return orderFields.get(o.entry, name) }

type orderSource struct{ in, not_in string }

var orderFields = fieldTable[OrderEntry]{
	"subject_uid":         func(o OrderEntry) float64 { return float64(o.SubjectUID) },
	"subject_kind":        func(o OrderEntry) float64 { return float64(o.SubjectKind) },
	"order_kind":          func(o OrderEntry) float64 { return float64(o.Kind) },
	"queue_position":      func(o OrderEntry) float64 { return float64(o.QueuePosition) },
	"target_uid":          func(o OrderEntry) float64 { return float64(o.TargetUID) },
	"has_target_location": func(o OrderEntry) float64 { return boolNumber(o.HasTargetLocation) },
	"has_target_zone":     func(o OrderEntry) float64 { return boolNumber(o.HasTargetZone) },
	"target_space_x":      func(o OrderEntry) float64 { return float64(o.Target.Space.X) },
	"target_space_y":      func(o OrderEntry) float64 { return float64(o.Target.Space.Y) },
	"target_zone_x":       func(o OrderEntry) float64 { return float64(o.Target.Zone.X) },
	"target_zone_y":       func(o OrderEntry) float64 { return float64(o.Target.Zone.Y) },
	"item_unit_id":        func(o OrderEntry) float64 { return float64(o.ItemUnitID) },
	"item_building_id":    func(o OrderEntry) float64 { return float64(o.ItemBuildingID) },
	"item_tech_id":        func(o OrderEntry) float64 { return float64(o.ItemTechID) },
}

func (s *orderSource) static() bool { return true }
func parseOrderSource(where map[string]any) (loopSource, error) {
	if err := checkKeys(where, "subject_in_set", "subject_not_in_set"); err != nil {
		return nil, err
	}
	s := &orderSource{}
	for key, target := range map[string]*string{"subject_in_set": &s.in, "subject_not_in_set": &s.not_in} {
		if raw, present := where[key]; present {
			name, ok := raw.(string)
			if !ok || !validVarName(name) {
				return nil, fmt.Errorf("%s needs a valid set name", key)
			}
			*target = name
		}
	}
	return s, nil
}
func (s *orderSource) items(view View, _ *Internals) []loopItem {
	entries := view.OrderQueue()
	items := make([]loopItem, len(entries))
	for i, entry := range entries {
		items[i] = orderItem{entry: entry}
	}
	return items
}

func (s *orderSource) validSets(i *Internals, c *census) bool {
	for _, name := range []string{s.in, s.not_in} {
		if set := i.setNamed(name); name != "" && set != nil && set.kind != "units" && set.kind != "buildings" {
			i.warnOnce("order_subject_set:"+name, fmt.Sprintf("ai: subject set %q must contain units or buildings", name))
			c.invalid = true
		}
	}
	return !c.invalid
}

func (s *orderSource) allows(subject uint64, i *Internals) bool {
	contains := func(name string) bool {
		if set := i.setNamed(name); set != nil {
			for _, key := range set.keys {
				if key.id == subject {
					return true
				}
			}
		}
		return false
	}
	return (s.in == "" || contains(s.in)) && (s.not_in == "" || !contains(s.not_in))
}
