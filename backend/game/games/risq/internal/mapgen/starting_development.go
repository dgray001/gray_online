package mapgen

import (
	"encoding/json"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func applyStartingDevelopmentDefaults(ctx *mapScriptContext, p *playerStartsParams) {
	if p.Assets != nil {
		p.Units, p.Buildings = p.Assets.Units, p.Assets.Buildings
		return
	}
	if ctx.vars["starting_development"] == float64(defs.StartingDevelopment_ADVANCED) {
		applyAdvancedDefaults(p)
		return
	}
	if ctx.vars["starting_development"] == float64(defs.StartingDevelopment_ESTABLISHED) {
		applyEstablishedDefaults(p)
		return
	}
	nomad := ctx.vars["starting_development"] == float64(defs.StartingDevelopment_NOMAD)
	if p.Units == nil {
		p.Units = []playerStartUnitJSON{{UnitId: 1, Count: ScriptExpr{raw: json.RawMessage("3")}, Target: zoneTargetJSON{Zone: "center"}}}
		if !nomad {
			p.Units = append(p.Units, playerStartUnitJSON{UnitId: 11, Count: ScriptExpr{raw: json.RawMessage("1")}, Target: zoneTargetJSON{Zone: "center"}})
		}
	}
	if p.Buildings == nil && !nomad {
		p.Buildings = []playerStartBuildingJSON{{BuildingId: 1, TerrainOverride: 7, Target: zoneTargetJSON{Zone: "center"}}}
	}
}
