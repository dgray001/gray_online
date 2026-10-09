package risq_mapgen_tests

import (
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func TestMalformedScriptsFail(t *testing.T) {
	cases := map[string]struct{ script, want string }{
		"empty":               {`[]`, "must start with a shape step"},
		"not_shape":           {`[{"step":"terrain_fill","params":{"terrain_id":1}}]`, "must start with a shape step"},
		"two_shapes":          {`[` + hexagon2 + `,` + hexagon2 + `]`, "first step"},
		"unknown_step":        {`[` + hexagon2 + `,{"step":"nope","params":{}}]`, `unknown map script step "nope"`},
		"bad_json":            {`{bad`, "failed to parse map script"},
		"bad_kind":            {shapeScript(`{"kind":"blob"}`), "unknown kind"},
		"negative_size":       {shapeScript(`{"kind":"hexagon","size":{"radius":-1}}`), "must be >= 0"},
		"ring_inner":          {shapeScript(`{"kind":"ring","size":{"outer_radius":3,"inner_radius":3}}`), "inner_size < size"},
		"rectangle_0":         {shapeScript(`{"kind":"rectangle","size":{"rows":0,"cols":2}}`), "rows and cols must be >= 1"},
		"bad_expr":            {shapeScript(`{"kind":"hexagon","size":{"radius":"1 +"}}`), "unexpected token"},
		"unknown_var":         {shapeScript(`{"kind":"hexagon","size":{"radius":"nope"}}`), `unknown variable "nope"`},
		"object_size":         {shapeScript(`{"kind":"hexagon","size":{"a":1}}`), "unknown hexagon size field"},
		"regions_rect":        {`[{"step":"shape","params":{"kind":"rectangle","size":{"cols":3,"rows":3}}},{"step":"regions_seven","params":{}}]`, "only valid on hexagon or ring shapes"},
		"regions_six_shape":   {`[` + hexagon2 + `,{"step":"regions_six","params":{}}]`, "only valid on rectangle shapes"},
		"regions_six_small":   {`[{"step":"shape","params":{"kind":"rectangle","size":{"rows":1,"cols":2}}},{"step":"regions_six","params":{}}]`, "at least 2 rows and 3 columns"},
		"regions_six_expr":    {`[{"step":"shape","params":{"kind":"rectangle","size":{"rows":4,"cols":6}}},{"step":"regions_six","params":{"center_bonus":"nope"}}]`, `unknown variable "nope"`},
		"regions_four_shape":  {`[` + hexagon2 + `,{"step":"regions_four","params":{}}]`, "only valid on triangle shapes"},
		"regions_four_small":  {`[{"step":"shape","params":{"kind":"triangle","size":{"edge_length":1}}},{"step":"regions_four","params":{}}]`, "edge_length of at least 2"},
		"regions_four_expr":   {`[{"step":"shape","params":{"kind":"triangle","size":{"edge_length":6}}},{"step":"regions_four","params":{"center_bonus":"nope"}}]`, `unknown variable "nope"`},
		"unknown_tech":        {`[` + hexagon2 + `,{"step":"rules","params":{"starting_techs":[999]}}]`, "unknown starting tech"},
		"center_bonus_expr":   {`[` + hexagon2 + `,{"step":"regions_seven","params":{"center_bonus":"nope"}}]`, `unknown variable "nope"`},
		"outer_bonus_expr":    {`[` + hexagon2 + `,{"step":"regions_seven","params":{"outer_bonus":"nope"}}]`, `unknown variable "nope"`},
		"scalar_size":         {shapeScript(`{"kind":"hexagon","size":2}`), "cannot unmarshal"},
		"removed_recommended": {shapeScript(`{"kind":"hexagon","size":{"radius":"recommended"}}`), `unknown variable "recommended"`},
		"no_starts":           {shapeScript(`{"kind":"hexagon","size":{"radius":2}}`), startsMissing},
		"bad_pattern": {`[{"step":"shape","params":{"kind":"hexagon","size":{"radius":4}}},
			{"step":"player_starts","params":{"pattern":"zig","area_size":1,"starting_distance":2,"resources":[],"buildings":[]}}]`, "unsupported player_starts pattern"},
	}
	scripts := map[string]string{}
	for name, c := range cases {
		scripts[name] = c.script
	}
	useScripts(t, scripts)
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := fakeboard.Generate("script:"+name, 2, 1); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("fake error %v, want it to contain %q", err, c.want)
			}
			if _, err := engineStarts("script:"+name, 2, 1); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("engine error %v, want it to contain %q", err, c.want)
			}
		})
	}
}
