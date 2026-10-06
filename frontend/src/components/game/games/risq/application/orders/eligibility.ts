import type { RisqBuilding, RisqFrontendOrder, RisqPlayer, RisqProducible, RisqUnit } from '../../model/types';
import { RisqOrderType, RisqProducibleKind, RisqUnitType, meetsTechRequirement } from '../../model/types';

export function producibleUnlocked(player: RisqPlayer, producible: RisqProducible): boolean {
  return (
    meetsTechRequirement(player, producible.required_tech_id) &&
    (producible.kind !== RisqProducibleKind.TECH || !player.researched_techs.get(producible.id))
  );
}

export function buildingCanProduce(player: RisqPlayer, building: RisqBuilding, producible: RisqProducible): boolean {
  return (
    building.player_id === player.player.player_id &&
    !building.under_construction &&
    building.produces.some((p: RisqProducible): boolean => p.kind === producible.kind && p.id === producible.id) &&
    producibleUnlocked(player, producible)
  );
}

export function unitCanBuild(player: RisqPlayer, unit: RisqUnit, building_id: number): boolean {
  return (
    unit.player_id === player.player.player_id &&
    unit.unit_type === RisqUnitType.ECONOMIC &&
    unit.builds.some((p: RisqProducible): boolean => p.id === building_id && producibleUnlocked(player, p))
  );
}

export function researchQueued(player: RisqPlayer, orders: RisqFrontendOrder[], tech_id: number): boolean {
  return (
    player.researched_techs.has(tech_id) ||
    orders.some(
      (o: RisqFrontendOrder): boolean =>
        o.order_type === RisqOrderType.OrderType_BuildingResearch && o.target_id === tech_id
    )
  );
}
