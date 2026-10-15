package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
)

type SpaceCondition struct {
	All              []SpaceCondition `json:"all,omitempty"`
	Any              []SpaceCondition `json:"any,omitempty"`
	Not              *SpaceCondition  `json:"not,omitempty"`
	Vision           *uint8           `json:"vision,omitempty"`
	VisionAtLeast    *uint8           `json:"vision_at_least,omitempty"`
	VisionAtMost     *uint8           `json:"vision_at_most,omitempty"`
	ResourceIDs      []uint32         `json:"resource_ids,omitempty"`
	BuildingIDs      []uint32         `json:"building_ids,omitempty"`
	Owner            string           `json:"owner,omitempty"`
	PlayerID         *int             `json:"player_id,omitempty"`
	BuildingOwner    string           `json:"building_owner,omitempty"`
	BuildingPlayerID *int             `json:"building_player_id,omitempty"`
	UnitCountAtLeast *int             `json:"unit_count_at_least,omitempty"`
	UnitCountAtMost  *int             `json:"unit_count_at_most,omitempty"`
}

func parseWhere(raw map[string]any) (SpaceCondition, error) {
	value, present := raw["where"]
	if !present {
		return SpaceCondition{}, nil
	}
	if value == nil {
		return SpaceCondition{}, fmt.Errorf("where must be an object")
	}
	return parseSpaceCondition(value)
}

func parseSpaceCondition(raw any) (SpaceCondition, error) {
	var condition SpaceCondition
	if raw == nil {
		return condition, nil
	}
	obj, ok := raw.(map[string]any)
	if !ok {
		return condition, fmt.Errorf("space condition must be an object")
	}
	for key, value := range obj {
		if value == nil {
			return condition, fmt.Errorf("space condition %q must not be null", key)
		}
		if key == "resource_ids" || key == "building_ids" {
			if _, err := parseIDSet(obj, key); err != nil {
				return condition, err
			}
		}
		if key == "owner" || key == "building_owner" {
			if owner, ok := value.(string); !ok || owner == "" {
				return condition, fmt.Errorf("%q must be an owner name", key)
			}
		}
		if key == "all" || key == "any" {
			children, ok := value.([]any)
			if !ok {
				return condition, fmt.Errorf("%q must be an array", key)
			}
			for _, child := range children {
				if _, ok := child.(map[string]any); !ok {
					return condition, fmt.Errorf("%q children must be objects", key)
				}
				if _, err := parseSpaceCondition(child); err != nil {
					return condition, err
				}
			}
		}
		if key == "not" {
			if _, ok := value.(map[string]any); !ok {
				return condition, fmt.Errorf("not must be an object")
			}
			if _, err := parseSpaceCondition(value); err != nil {
				return condition, err
			}
		}
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return condition, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&condition); err != nil {
		return condition, err
	}
	return condition, condition.validate()
}

func (c SpaceCondition) validate() error {
	for _, vision := range []*uint8{c.Vision, c.VisionAtLeast, c.VisionAtMost} {
		if vision != nil && *vision > 4 {
			return fmt.Errorf("vision must be between 0 and 4")
		}
	}
	for _, owner := range []string{c.Owner, c.BuildingOwner} {
		if owner != "" && owner != "own" && owner != "enemy" && owner != "unowned" && owner != "any" {
			return fmt.Errorf("owner must be own, enemy, unowned or any")
		}
	}
	for _, child := range append(append([]SpaceCondition{}, c.All...), c.Any...) {
		if err := child.validate(); err != nil {
			return err
		}
	}
	if c.Not != nil {
		return c.Not.validate()
	}
	return nil
}

type spaceQuery struct {
	condition         SpaceCondition
	from              anchor
	has_from          bool
	direction_from    anchor
	direction         int
	exclude_reference bool
	// only spaces holding at least one unit of this bucket
	in_bucket string
}

func parseSpaceQuery(obj map[string]any) (*spaceQuery, error) {
	q := &spaceQuery{has_from: obj["from"] != nil}
	var err error
	if q.condition, err = parseWhere(obj); err != nil {
		return nil, err
	}
	if q.from, err = parseAnchor(obj["from"]); err != nil {
		return nil, err
	}
	if raw, present := obj["direction_from"]; present {
		if q.direction_from, err = parseAnchor(raw); err != nil {
			return nil, err
		}
		name, ok := obj["direction"].(string)
		if !ok || name != "away" && name != "toward" {
			return nil, fmt.Errorf("direction must be away or toward")
		}
		q.direction = 1
		if name == "away" {
			q.direction = -1
		}
	}
	if _, present := obj["direction_away_from"]; present {
		q.direction = -1
		q.direction_from, err = parseAnchor(obj["direction_away_from"])
		if err != nil {
			return nil, err
		}
	}
	q.in_bucket, _ = obj["held_by_bucket"].(string)
	if raw, present := obj["exclude_reference"]; present {
		var ok bool
		if q.exclude_reference, ok = raw.(bool); !ok {
			return nil, fmt.Errorf("exclude_reference must be a bool")
		}
	}
	return q, nil
}

func (q *spaceQuery) resolve(view View, internals *Internals, from *ZoneRef) (ZoneRef, func(Coordinate) bool, bool) {
	var reference ZoneRef
	var ok bool
	if from != nil && (!q.has_from || q.from.kind == anchorUnit) {
		reference, ok = *from, true
	} else {
		reference, ok = q.from.location(view, internals)
	}
	if !ok {
		return ZoneRef{}, nil, false
	}
	include := func(space Coordinate) bool { return true }
	if q.in_bucket != "" {
		held := map[Coordinate]bool{}
		for _, u := range view.Units() {
			if internals.inBucket(q.in_bucket, u.InternalID) {
				held[u.Location.Space] = true
			}
		}
		include = func(space Coordinate) bool { return held[space] }
	}
	if q.direction != 0 {
		away, found := q.direction_from.location(view, internals)
		if !found {
			return ZoneRef{}, nil, false
		}
		x, y := away.Space.X-reference.Space.X, away.Space.Y-reference.Space.Y
		held := include
		include = func(space Coordinate) bool {
			a, b := space.X-reference.Space.X, space.Y-reference.Space.Y
			return held(space) && q.direction*(a*x+b*y+(a+b)*(x+y)) > 0
		}
	}
	return reference, include, true
}

func (q *spaceQuery) buildFilter(view View, internals *Internals) (ZoneRef, func(Coordinate) bool, bool) {
	reference, direction, ok := q.resolve(view, internals, nil)
	if !ok {
		return ZoneRef{}, nil, false
	}
	spaces := make(map[Coordinate]bool)
	view.CountSpaces(q.condition, func(space Coordinate) bool {
		if direction(space) && (!q.exclude_reference || space != reference.Space) {
			spaces[space] = true
		}
		return false
	})
	return reference, func(space Coordinate) bool { return spaces[space] }, true
}

func (q *spaceQuery) closest(view View, internals *Internals, from *ZoneRef) []Coordinate {
	reference, include, ok := q.resolve(view, internals, from)
	if !ok {
		return nil
	}
	if q.direction != 0 || q.in_bucket != "" {
		var nearest []Coordinate
		best := -1
		view.CountSpaces(q.condition, func(space Coordinate) bool {
			if !include(space) || q.exclude_reference && space == reference.Space {
				return false
			}
			d := view.SpaceDistance(reference.Space, space)
			if best < 0 || d < best {
				nearest, best = []Coordinate{space}, d
			} else if d == best {
				nearest = append(nearest, space)
			}
			return false
		})
		return nearest
	}
	return view.ClosestSpaces(reference.Space, q.condition, q.exclude_reference)
}

func parseClosestSpaceCounter(obj map[string]any) (counter, error) {
	query, err := parseSpaceQuery(obj)
	if err != nil {
		return nil, err
	}
	return func(view View, internals *Internals) float64 {
		return float64(len(query.closest(view, internals, nil)))
	}, nil
}

func parseSpaceCounter(obj map[string]any) (counter, error) {
	condition, err := parseWhere(obj)
	if err != nil {
		return nil, err
	}
	near, err := parseNearFilter(obj)
	if err != nil {
		return nil, err
	}
	return func(view View, internals *Internals) float64 {
		return float64(view.CountSpaces(condition, func(space Coordinate) bool {
			return near.contains(view, internals, ZoneRef{Space: space})
		}))
	}, nil
}

// Where a "within" distance is measured from: our home (default), the center of a bucket's units, or the
// candidate currently being scored by a target picker.
type anchorKind uint8

const (
	anchorHome anchorKind = iota
	anchorBucket
	anchorTarget
	// the unit a "unit_when" condition is being checked for
	anchorUnit
	anchorCoordinate
)

type anchor struct {
	kind                       anchorKind
	bucket                     string
	coordinate_x, coordinate_y amount
}

func parseAnchor(raw any) (anchor, error) {
	if raw == nil {
		return anchor{kind: anchorHome}, nil
	}
	if obj, ok := raw.(map[string]any); ok {
		x, x_err := parseAmount(obj["x"])
		y, y_err := parseAmount(obj["y"])
		if x_err != nil || y_err != nil {
			return anchor{}, fmt.Errorf("coordinate reference requires x and y as numbers or expressions")
		}
		return anchor{kind: anchorCoordinate, coordinate_x: x, coordinate_y: y}, nil
	}
	name, ok := raw.(string)
	if !ok || name == "" {
		return anchor{}, fmt.Errorf("\"from\" must be a coordinate object, \"home\", \"target\", \"unit\" or a bucket name")
	}
	switch name {
	case "home":
		return anchor{kind: anchorHome}, nil
	case "target":
		return anchor{kind: anchorTarget}, nil
	case "unit":
		return anchor{kind: anchorUnit}, nil
	}
	return anchor{kind: anchorBucket, bucket: name}, nil
}

func (a anchor) location(view View, internals *Internals) (ZoneRef, bool) {
	switch a.kind {
	case anchorCoordinate:
		return ZoneRef{Space: Coordinate{X: a.coordinate_x.int(view, internals), Y: a.coordinate_y.int(view, internals)}}, true
	case anchorBucket:
		b := internals.Buckets[a.bucket]
		if b == nil {
			return ZoneRef{}, false
		}
		members := make([]UnitView, 0, len(b.Members))
		for _, u := range view.Units() {
			if b.Members[u.InternalID] {
				members = append(members, u)
			}
		}
		return groupCenter(view, members)
	case anchorTarget:
		if internals.target == nil {
			return ZoneRef{}, false
		}
		return *internals.target, true
	case anchorUnit:
		if internals.unit == nil {
			return ZoneRef{}, false
		}
		return *internals.unit, true
	}
	return homeLocation(view)
}

// The unit closest to all the others, so a group's center is always a real, reachable zone
func groupCenter(view View, units []UnitView) (ZoneRef, bool) {
	if len(units) == 0 {
		return ZoneRef{}, false
	}
	best, best_sum := units[0].Location, -1
	for _, u := range units {
		sum := 0
		for _, other := range units {
			sum += view.SpaceDistance(u.Location.Space, other.Location.Space)
		}
		if best_sum < 0 || sum < best_sum {
			best, best_sum = u.Location, sum
		}
	}
	return best, true
}

// Optional "within" (spaces) around an anchor; without "within" everything counts
type nearFilter struct {
	within     int
	has_within bool
	from       anchor
}

func parseNearFilter(obj map[string]any) (nearFilter, error) {
	f := nearFilter{}
	if value, present := obj["within"]; present {
		w, ok := value.(float64)
		if !ok || w < 0 || w != math.Trunc(w) || math.IsInf(w, 0) {
			return f, fmt.Errorf("within must be a finite nonnegative whole number")
		}
		f.within, f.has_within = int(w), true
	}
	if value, present := obj["from"]; present && value == nil {
		return f, fmt.Errorf("from must not be null")
	}
	from, err := parseAnchor(obj["from"])
	if err != nil {
		return f, err
	}
	if obj["from"] != nil && !f.has_within {
		return f, fmt.Errorf("\"from\" needs a \"within\" distance")
	}
	f.from = from
	return f, nil
}

func (f nearFilter) contains(view View, internals *Internals, loc ZoneRef) bool {
	if !f.has_within {
		return true
	}
	center, ok := f.from.location(view, internals)
	return ok && view.SpaceDistance(center.Space, loc.Space) <= f.within
}
