import type { RisqBuilding, RisqPlayer, RisqCost } from './types';
import { RisqProducibleKind, RisqResourceType } from './types';
export function canHaveGatherPoint(building: RisqBuilding): boolean {
  return building.garrison_capacity > 0 || building.produces.some((p) => p.kind === RisqProducibleKind.UNIT);
}

/** Returns whether the player can currently afford the input cost, accounting for already-queued spending */
export function canAffordCost(player: RisqPlayer, cost: RisqCost): boolean {
  const net = (type: RisqResourceType) => {
    const pr = player.resources.get(type);
    return pr ? pr.amount - pr.spending : 0;
  };
  return (
    net(RisqResourceType.FOOD) >= cost.food &&
    net(RisqResourceType.WOOD) >= cost.wood &&
    net(RisqResourceType.STONE) >= cost.stone &&
    net(RisqResourceType.GOLD) >= cost.gold
  );
}

export function meetsTechRequirement(player: RisqPlayer, required_tech_id: number): boolean {
  return required_tech_id === 0 || !!player.researched_techs.get(required_tech_id);
}
