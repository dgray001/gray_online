package bridge

import "github.com/dgray001/gray_online/game/games/risq/ai"

type observation struct {
	Units            []ai.UnitView
	VisibleBuildings []ai.BuildingView
	KnownBuildings   []ai.BuildingView
	Resources        []ai.ResourceView
	Spaces           []ai.SpaceInfo
}

type recordingModel struct {
	ai.Model
	Seen chan observation
}

func (m *recordingModel) DecideOrders(view ai.View) ai.Decision {
	m.Seen <- observation{Units: view.VisibleEnemyUnits(), VisibleBuildings: view.VisibleEnemyBuildings(), KnownBuildings: view.KnownEnemyBuildings(), Resources: view.KnownResources(ai.ResourceFood), Spaces: view.AllSpaces()}
	return m.Model.DecideOrders(view)
}
