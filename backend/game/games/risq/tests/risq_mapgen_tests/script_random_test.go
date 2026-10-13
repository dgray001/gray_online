package risq_mapgen_tests

import (
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func biomeFillScript() string {
	define := step("define", `{"name":"biome","value":"randomInt(0, 2)"}`)
	return `[` + hexagon3 + `,` + define + `,` + step("terrain_fill", `{"terrain_id":"1 + biome"}`) + `]`
}

func TestScriptRandomIntDrivesAnExpressionTerrainId(t *testing.T) {
	useScripts(t, map[string]string{"biome": biomeFillScript()})
	seen := map[uint32]bool{}
	for seed := int64(1); seed <= 40; seed++ {
		board, _ := fakeboard.Generate("script:biome", 2, seed)
		for _, id := range []uint32{1, 2, 3} {
			if len(withTerrain(board, id)) == 37 {
				seen[id] = true
			}
		}
	}
	if len(seen) != 3 {
		t.Errorf("whole-board terrains %v across seeds, want 1, 2 and 3 each reached", seen)
	}
}

func TestScriptRandomIntIsReproduciblePerSeed(t *testing.T) {
	useScripts(t, map[string]string{"biome": biomeFillScript()})
	for seed := int64(1); seed <= 10; seed++ {
		first, _ := fakeboard.Generate("script:biome", 2, seed)
		second, _ := fakeboard.Generate("script:biome", 2, seed)
		for _, id := range []uint32{1, 2, 3} {
			if len(withTerrain(first, id)) != len(withTerrain(second, id)) {
				t.Fatalf("seed %d painted terrain %d differently on two runs", seed, id)
			}
		}
	}
}

func TestExpressionTerrainIdMustNameAKnownTerrain(t *testing.T) {
	define := step("define", `{"name":"bad","value":9999}`)
	useScripts(t, map[string]string{"bad": `[` + hexagon3 + `,` + define + `,` + step("terrain_fill", `{"terrain_id":"bad"}`) + `]`})
	if _, err := fakeboard.Generate("script:bad", 2, 1); err == nil || !strings.Contains(err.Error(), "unknown terrain id 9999") {
		t.Errorf("got %v, want an unknown terrain id error", err)
	}
}

func TestTerrainIdsWeightsAcceptExpressions(t *testing.T) {
	define := step("define", `{"name":"w","value":0}`)
	ids := `{"terrain_ids":[{"terrain_id":2,"weight":"w"},{"terrain_id":"1 + w","weight":"1 + w"}]}`
	useScripts(t, map[string]string{"weights": `[` + hexagon3 + `,` + define + `,` + step("terrain_fill", ids) + `]`})
	for seed := int64(1); seed <= 10; seed++ {
		board, _ := fakeboard.Generate("script:weights", 2, seed)
		if got := len(withTerrain(board, 1)); got != 37 {
			t.Fatalf("seed %d: %d spaces took the weighted terrain, want all 37 since the other weight is 0", seed, got)
		}
	}
}

func widestBlobRadius(t *testing.T, roundness string) int {
	t.Helper()
	blob := `{"terrain_id":21,"seed_count":1,"size":19,"roundness":` + roundness + `,"min_spacing":0,"placement":{"pattern":"ring","distance":[0,0]}}`
	useScripts(t, map[string]string{"round": blobScript(hexagon4, blob)})
	widest := 0
	for seed := int64(1); seed <= 30; seed++ {
		board, _ := fakeboard.Generate("script:round", 2, seed)
		for _, c := range withTerrain(board, 21) {
			widest = max(widest, int(game_utils.AxialDistance(game_utils.Coordinate2D{}, c)))
		}
	}
	return widest
}

func TestFullRoundnessGrowsAnExactBallAndZeroLeaksPastIt(t *testing.T) {
	if got := widestBlobRadius(t, "1"); got != 2 {
		t.Errorf("roundness 1 reached distance %d, want a 19 space ball of radius 2", got)
	}
	if got := widestBlobRadius(t, "0"); got <= 2 {
		t.Errorf("roundness 0 reached distance %d, want some blob to extend past radius 2", got)
	}
}

func TestIfBranchFollowsARandomDefine(t *testing.T) {
	define := step("define", `{"name":"biome","value":"randomInt(0, 1)"}`)
	pick := func(biome string) string {
		return step("if", `{"condition":"1 - (biome - `+biome+`) * (biome - `+biome+`)","then":[`+
			step("terrain_fill", `{"terrain_id":"`+biome+` + 2"}`)+`]}`)
	}
	useScripts(t, map[string]string{"pick": `[` + hexagon3 + `,` + define + `,` + pick("0") + `,` + pick("1") + `]`})
	for seed := int64(1); seed <= 20; seed++ {
		board, _ := fakeboard.Generate("script:pick", 2, seed)
		if two, three := len(withTerrain(board, 2)), len(withTerrain(board, 3)); two+three != 37 || two*three != 0 {
			t.Fatalf("seed %d: %d and %d spaces of terrain 2 and 3, want exactly one branch to paint all 37", seed, two, three)
		}
	}
}
