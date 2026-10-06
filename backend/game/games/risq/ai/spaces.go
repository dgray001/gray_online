package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func parseSpaceCondition(raw any) (SpaceCondition, error) {
	var condition SpaceCondition
	if raw == nil {
		return condition, nil
	}
	if _, ok := raw.(map[string]any); !ok {
		return condition, fmt.Errorf("space condition must be an object")
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
	if q.condition, err = parseSpaceCondition(obj["where"]); err != nil {
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
	reference, ok := q.from.location(view, internals)
	if from != nil && (!q.has_from || q.from.kind == anchorUnit) {
		reference, ok = *from, true
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
	condition, err := parseSpaceCondition(obj["where"])
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
