package ai

import (
	"fmt"
	"slices"
	"strings"
)

var assignmentFieldNames = []string{
	"actor_uid", "target_uid", "actor_x", "actor_y", "actor_zone_x", "actor_zone_y",
	"target_x", "target_y", "target_zone_x", "target_zone_y", "weight", "round",
}

type assignmentItem struct {
	index int
	rec   assignRecord
}

func assignedUID(item loopItem) (float64, bool) {
	switch kindOfItem(item) {
	case "spaces", "zones", "regions", "":
		return 0, false
	}
	key, _ := keyOf(item)
	return float64(key.id), true
}

func assignedCoordinate(item loopItem, name string) (float64, bool) {
	loc, _, located := locOf(item)
	if !located {
		return 0, false
	}
	switch name {
	case "x":
		return float64(loc.Space.X), true
	case "y":
		return float64(loc.Space.Y), true
	case "zone_x":
		return float64(loc.Zone.X), true
	}
	return float64(loc.Zone.Y), true
}

func (a assignmentItem) field(name string) (float64, bool) {
	switch name {
	case "weight":
		return a.rec.weight, true
	case "round":
		return float64(a.rec.round), true
	}
	side, rest, _ := strings.Cut(name, "_")
	item := a.rec.actor
	if side == "target" {
		item = a.rec.target
	}
	if rest == "uid" {
		return assignedUID(item)
	}
	return assignedCoordinate(item, rest)
}

type assignmentSource struct{ relation, actor_kind, target_kind string }

func parseAssignmentSource(where map[string]any) (loopSource, error) {
	if err := checkKeys(where, "relation", "actor_kind", "target_kind"); err != nil {
		return nil, err
	}
	name, ok := where["relation"].(string)
	if !ok || !validVarName(name) {
		return nil, fmt.Errorf("assignments need a valid \"relation\" name")
	}
	s := &assignmentSource{relation: name}
	for key, target := range map[string]*string{"actor_kind": &s.actor_kind, "target_kind": &s.target_kind} {
		if raw, present := where[key]; present {
			kind, ok := raw.(string)
			if !ok || kind == "assignments" || !slices.Contains(selectKinds, kind) {
				return nil, fmt.Errorf("%s must name a selector kind", key)
			}
			*target = kind
		}
	}
	return s, nil
}

func (s *assignmentSource) items(_ View, internals *Internals) []loopItem {
	job := internals.assignments[s.relation]
	if job == nil {
		internals.warnOnce("relation:"+s.relation, fmt.Sprintf("ai: no assignment named %q", s.relation))
		internals.eval_errors++
		return nil
	}
	items := make([]loopItem, 0, len(job.records))
	for index, rec := range job.records {
		if (s.actor_kind == "" || rec.actor_key.kind == s.actor_kind) && (s.target_kind == "" || rec.target_key.kind == s.target_kind) {
			items = append(items, assignmentItem{index: index, rec: rec})
		}
	}
	return items
}
