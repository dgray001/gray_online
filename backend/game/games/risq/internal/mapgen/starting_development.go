package mapgen

import (
	"encoding/json"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

type startingDevIDs struct {
	worker, grunt, archer, heavy                       uint32
	townCenter, house, farm, research, tower, barracks uint32
}

func getStartingDevIDs() startingDevIDs {
	if _, ok := defs.UnitConfigs[1]; ok {
		return startingDevIDs{1, 101, 102, 103, 1, 2, 3, 11, 21, 22}
	}
	return startingDevIDs{11, 21, 31, 22, 201, 202, 203, 209, 207, 205}
}

func applyStartingDevelopmentDefaults(ctx *mapScriptContext, p *playerStartsParams) {
	if p.Assets != nil {
		p.Units, p.Buildings = p.Assets.Units, p.Assets.Buildings
		return
	}
	ids := getStartingDevIDs()
	if ctx.vars["starting_development"] == float64(defs.StartingDevelopment_ADVANCED) {
		applyAdvancedDefaults(p, ids)
		return
	}
	if ctx.vars["starting_development"] == float64(defs.StartingDevelopment_ESTABLISHED) {
		applyEstablishedDefaults(p, ids)
		return
	}
	nomad := ctx.vars["starting_development"] == float64(defs.StartingDevelopment_NOMAD)
	if p.Units == nil {
		p.Units = []playerStartUnitJSON{{UnitId: ids.worker, Count: ScriptExpr{raw: json.RawMessage("3")}, Target: zoneTargetJSON{Zone: "center"}}}
		if !nomad {
			p.Units = append(p.Units, playerStartUnitJSON{UnitId: ids.grunt, Count: ScriptExpr{raw: json.RawMessage("1")}, Target: zoneTargetJSON{Zone: "center"}})
		}
	}
	if p.Buildings == nil && !nomad {
		p.Buildings = []playerStartBuildingJSON{{BuildingId: ids.townCenter, TerrainOverride: 7, Target: zoneTargetJSON{Zone: "center"}}}
	}
}

func applyEstablishedDefaults(p *playerStartsParams, ids startingDevIDs) {
	if p.Units == nil {
		p.Units = []playerStartUnitJSON{
			{UnitId: ids.worker, Count: ScriptExpr{raw: json.RawMessage("6")}, Target: zoneTargetJSON{Zone: "center"}},
			{UnitId: ids.grunt, Count: ScriptExpr{raw: json.RawMessage("2")}, Target: zoneTargetJSON{Zone: "center"}},
			{UnitId: ids.archer, Count: ScriptExpr{raw: json.RawMessage("1")}, Target: zoneTargetJSON{Zone: "center"}},
		}
	}
	if p.Buildings == nil {
		p.Buildings = []playerStartBuildingJSON{
			{BuildingId: ids.townCenter, TerrainOverride: 7, Target: zoneTargetJSON{Zone: "center"}},
			{BuildingId: ids.farm, TerrainOverride: 25, Target: zoneTargetJSON{Zone: "edge"}},
			{BuildingId: ids.tower, Target: zoneTargetJSON{SpaceDistance: 1, Zone: "center"}},
			{BuildingId: ids.house, Target: zoneTargetJSON{SpaceDistance: 1, Zone: "center"}},
			{BuildingId: ids.house, Target: zoneTargetJSON{SpaceDistance: 1, Zone: "center"}},
		}
	}
}

func applyAdvancedDefaults(p *playerStartsParams, ids startingDevIDs) {
	if p.Units == nil {
		p.Units = []playerStartUnitJSON{
			{UnitId: ids.worker, Count: ScriptExpr{raw: json.RawMessage("10")}, Target: zoneTargetJSON{Zone: "center"}},
			{UnitId: ids.grunt, Count: ScriptExpr{raw: json.RawMessage("4")}, Target: zoneTargetJSON{Zone: "center"}},
			{UnitId: ids.archer, Count: ScriptExpr{raw: json.RawMessage("2")}, Target: zoneTargetJSON{Zone: "center"}},
			{UnitId: ids.heavy, Count: ScriptExpr{raw: json.RawMessage("1")}, Target: zoneTargetJSON{Zone: "center"}},
		}
	}
	if p.Buildings == nil {
		p.Buildings = []playerStartBuildingJSON{
			{BuildingId: ids.townCenter, TerrainOverride: 7, Target: zoneTargetJSON{Zone: "center"}},
			{BuildingId: ids.farm, TerrainOverride: 25, Target: zoneTargetJSON{Zone: "edge"}},
			{BuildingId: ids.farm, TerrainOverride: 25, Target: zoneTargetJSON{Zone: "edge"}},
			{BuildingId: ids.farm, TerrainOverride: 25, Target: zoneTargetJSON{Zone: "edge"}},
			{BuildingId: ids.farm, TerrainOverride: 25, Target: zoneTargetJSON{Zone: "edge"}},
			{BuildingId: ids.house, Target: zoneTargetJSON{Zone: "edge"}},
			{BuildingId: ids.house, Target: zoneTargetJSON{Zone: "edge"}},
			{BuildingId: ids.house, Target: zoneTargetJSON{SpaceDistance: 1, Zone: "center"}},
			{BuildingId: ids.house, Target: zoneTargetJSON{SpaceDistance: 1, Zone: "center"}},
			{BuildingId: ids.tower, Target: zoneTargetJSON{SpaceDistance: 1, Zone: "center"}},
			{BuildingId: ids.research, Target: zoneTargetJSON{SpaceDistance: 1, Zone: "center"}},
			{BuildingId: ids.barracks, Target: zoneTargetJSON{SpaceDistance: 1, Zone: "center"}},
		}
	}
}
