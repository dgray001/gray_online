package mapgen

import "encoding/json"

func applyAdvancedDefaults(p *playerStartsParams) {
	if p.Units == nil {
		p.Units = []playerStartUnitJSON{
			{UnitId: 1, Count: ScriptExpr{raw: json.RawMessage("10")}, Target: zoneTargetJSON{Zone: "center"}},
			{UnitId: 11, Count: ScriptExpr{raw: json.RawMessage("4")}, Target: zoneTargetJSON{Zone: "center"}},
			{UnitId: 12, Count: ScriptExpr{raw: json.RawMessage("2")}, Target: zoneTargetJSON{Zone: "center"}},
			{UnitId: 13, Count: ScriptExpr{raw: json.RawMessage("1")}, Target: zoneTargetJSON{Zone: "center"}},
		}
	}
	if p.Buildings == nil {
		p.Buildings = []playerStartBuildingJSON{
			{BuildingId: 1, TerrainOverride: 7, Target: zoneTargetJSON{Zone: "center"}},
			{BuildingId: 3, TerrainOverride: 25, Target: zoneTargetJSON{Zone: "edge"}},
			{BuildingId: 3, TerrainOverride: 25, Target: zoneTargetJSON{Zone: "edge"}},
			{BuildingId: 3, TerrainOverride: 25, Target: zoneTargetJSON{Zone: "edge"}},
			{BuildingId: 3, TerrainOverride: 25, Target: zoneTargetJSON{Zone: "edge"}},
			{BuildingId: 2, Target: zoneTargetJSON{Zone: "edge"}},
			{BuildingId: 2, Target: zoneTargetJSON{Zone: "edge"}},
			{BuildingId: 2, Target: zoneTargetJSON{SpaceDistance: 1, Zone: "center"}},
			{BuildingId: 2, Target: zoneTargetJSON{SpaceDistance: 1, Zone: "center"}},
			{BuildingId: 21, Target: zoneTargetJSON{SpaceDistance: 1, Zone: "center"}},
			{BuildingId: 11, Target: zoneTargetJSON{SpaceDistance: 1, Zone: "center"}},
			{BuildingId: 22, Target: zoneTargetJSON{SpaceDistance: 1, Zone: "center"}},
		}
	}
}
