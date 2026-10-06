package risq_mapgen_tests

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

type zoneView struct {
	Resource uint32
	Building [2]uint32
	Units    map[[2]uint32]int
	Override uint32
}

type spaceView struct {
	Terrain uint32
	Zones   map[string]zoneView
}

func num(m map[string]any, key string) uint32 {
	n, _ := m[key].(float64)
	return uint32(n)
}

func point(m map[string]any) string {
	c, _ := m["coordinate"].(map[string]any)
	return fmt.Sprintf("%v,%v", c["x"], c["y"])
}

type previewData struct {
	Spaces  map[string]spaceView
	Links   []string
	Regions []string
	Incomes map[float64]bool
}

// Everything the engine built for a custom map document that a map script or loader decides
func enginePreview(t *testing.T, doc string) (previewData, error) {
	t.Helper()
	state, err := risq.PreviewCustomMap([]byte(doc))
	if err != nil {
		return previewData{}, err
	}
	data, _ := json.Marshal(state)
	var parsed struct {
		Spaces     [][]map[string]any
		SpaceLinks []map[string]any `json:"space_links"`
		Regions    []struct {
			Name      string
			GoldBonus float64 `json:"gold_bonus"`
			Spaces    []uint
		}
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	result := previewData{Spaces: map[string]spaceView{}, Incomes: map[float64]bool{}, Links: engineLinks(parsed.SpaceLinks)}
	for _, row := range parsed.Spaces {
		for _, space := range row {
			if space != nil {
				result.Spaces[point(space)] = engineSpace(space)
				income, _ := space["gold_income"].(float64)
				result.Incomes[income] = true
			}
		}
	}
	for _, region := range parsed.Regions {
		slices.Sort(region.Spaces)
		result.Regions = append(result.Regions, fmt.Sprintf("%s|%v|%v", region.Name, region.GoldBonus, region.Spaces))
	}
	slices.Sort(result.Regions)
	return result, nil
}

func engineLinks(links []map[string]any) []string {
	lines := []string{}
	for _, link := range links {
		from, to := link["from"].(map[string]any), link["to"].(map[string]any)
		direction := "-"
		if d, ok := link["direction"].(float64); ok {
			direction = fmt.Sprint(d)
		}
		lines = append(lines, fmt.Sprintf("%v,%v>%v,%v:%s", from["x"], from["y"], to["x"], to["y"], direction))
	}
	slices.Sort(lines)
	return lines
}

func engineSpace(space map[string]any) spaceView {
	view := spaceView{Terrain: num(space, "terrain_id"), Zones: map[string]zoneView{}}
	rows, _ := space["zones"].([]any)
	for _, row := range rows {
		for _, raw := range row.([]any) {
			zone := raw.(map[string]any)
			z := zoneView{Units: map[[2]uint32]int{}, Override: num(zone, "terrain_override")}
			if r, ok := zone["resource"].(map[string]any); ok {
				z.Resource = num(r, "resource_id")
			}
			if b, ok := zone["building"].(map[string]any); ok {
				z.Building = [2]uint32{num(b, "building_id"), num(b, "player_id")}
			}
			units, _ := zone["units"].([]any)
			for _, u := range units {
				unit := u.(map[string]any)
				z.Units[[2]uint32{num(unit, "unit_id"), num(unit, "player_id")}]++
			}
			view.Zones[point(zone)] = z
		}
	}
	return view
}

func fakeViews(board *fakeboard.Board) map[string]spaceView {
	views := map[string]spaceView{}
	for _, info := range board.Inspect() {
		view := spaceView{Terrain: info.Terrain, Zones: map[string]zoneView{}}
		for _, z := range info.Zones {
			zone := zoneView{Resource: z.Resource, Building: [2]uint32{z.Building.ID, uint32(z.Building.Player)}, Units: map[[2]uint32]int{}, Override: z.Override}
			if z.Building.ID == 0 {
				zone.Building = [2]uint32{}
			}
			for _, u := range z.Units {
				zone.Units[[2]uint32{u.ID, uint32(u.Player)}]++
			}
			view.Zones[fmt.Sprintf("%d,%d", z.Local.X, z.Local.Y)] = zone
		}
		views[fmt.Sprintf("%d,%d", info.Coord.X, info.Coord.Y)] = view
	}
	return views
}
