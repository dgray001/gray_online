import { capitalize } from '../../../../../../../scripts/util';
import { equalsPoint2D } from '../../../../../util/objects2d';
import { invertZoneKey } from '../../../model/coordinates';
import type { RisqFrontendOrder, RisqSpace, RisqUnit } from '../../../model/types';
import { RisqOrderType, RisqResourceType, RisqUnitType } from '../../../model/types';
import { resourceType } from '../../../rendering/assets/resources';
import { unitCountLabel } from '../../../rendering/assets/unit';
import type { DwgRisq } from '../../../risq';
import { shortLabel } from '../../order_row/order_row_resolve';
import type { RisqTooltipData } from '../../risq_tooltip';

const ACTIVITIES: Partial<Record<RisqOrderType, string>> = {
  [RisqOrderType.OrderType_UnitMoveSpace]: 'moving',
  [RisqOrderType.OrderType_UnitMoveZone]: 'moving',
  [RisqOrderType.OrderType_UnitBuild]: 'building',
  [RisqOrderType.OrderType_UnitRepair]: 'repairing',
  [RisqOrderType.OrderType_UnitRenew]: 'renewing',
  [RisqOrderType.OrderType_UnitGarrison]: 'garrisoning',
  [RisqOrderType.OrderType_UnitUngarrison]: 'ungarrisoning',
  [RisqOrderType.OrderType_UnitAttackSpace]: 'attacking',
  [RisqOrderType.OrderType_UnitAttackZone]: 'attacking',
  [RisqOrderType.OrderType_UnitAttackUnit]: 'attacking',
  [RisqOrderType.OrderType_UnitAttackBuilding]: 'attacking',
  [RisqOrderType.OrderType_UnitAutoAttackUnit]: 'attacking',
  [RisqOrderType.OrderType_UnitAutoAttackBuilding]: 'attacking',
};

function gatherActivity(risq: DwgRisq, order: RisqFrontendOrder): string {
  const { space, zone } = invertZoneKey(order.target_id);
  const target = risq.session
    .spaceAt(space)
    ?.zones?.flat()
    .find((z) => equalsPoint2D(z.coordinate, zone));
  const type = target?.resource ? resourceType(target.resource) : target?.building?.resource_category;
  return type === undefined ? 'gathering' : `gathering ${RisqResourceType[type].toLowerCase()}`;
}

function activity(risq: DwgRisq, unit: RisqUnit): string {
  const order = risq.orders_model.effectiveForSubject(unit.internal_id, 'unit')[0];
  if (!order) {
    return 'idle';
  }
  if (order.order_type === RisqOrderType.OrderType_UnitGather) {
    return gatherActivity(risq, order);
  }
  return ACTIVITIES[order.order_type] ?? shortLabel(order.order_type).toLowerCase();
}

function tally<T>(items: T[], key: (item: T) => string): [string, number][] {
  const counts = new Map<string, number>();
  for (const item of items) {
    counts.set(key(item), (counts.get(key(item)) ?? 0) + 1);
  }
  return [...counts].sort((a, b) => b[1] - a[1]);
}

function spaceUnits(space: RisqSpace, economic: boolean): RisqUnit[] {
  return [...(space.units?.values() ?? [])].filter((u) => (u.unit_type === RisqUnitType.ECONOMIC) === economic);
}

function nickname(risq: DwgRisq, player_id: number): string {
  return risq.getGame()?.players[player_id]?.player.nickname ?? 'Unknown';
}

/** Your villagers by current activity, then other players' villager counts */
export function villagerTooltipData(risq: DwgRisq, space: RisqSpace): RisqTooltipData {
  const villagers = spaceUnits(space, true);
  const own = villagers.filter((u) => u.player_id === risq.getPlayerId());
  const others = villagers.filter((u) => u.player_id !== risq.getPlayerId());
  return {
    title: unitCountLabel('Villager', villagers.length),
    lines: [
      ...tally(own, (u) => activity(risq, u)).map(([label, count]) => `${count} ${label}`),
      ...tally(others, (u) => nickname(risq, u.player_id)).map(([name, count]) => `${name}: ${count}`),
    ],
  };
}

function typeLabel(type: RisqUnitType, count: number, owner?: string): string {
  const label = `${count} ${capitalize(RisqUnitType[type])} unit${count === 1 ? '' : 's'}`;
  return owner ? `${label} (${owner})` : label;
}

/** Military units by unit type, your own first and other players' marked with their name */
export function militaryTooltipData(risq: DwgRisq, space: RisqSpace): RisqTooltipData {
  const military = spaceUnits(space, false);
  const own_id = risq.getPlayerId();
  const lines = tally(military, (u) => `${u.unit_type}:${u.player_id}`)
    .map(([key, count]) => ({ ids: key.split(':').map(Number), count }))
    .sort((a, b) => Number(a.ids[1] !== own_id) - Number(b.ids[1] !== own_id))
    .map(({ ids: [type, player_id], count }) =>
      typeLabel(type, count, player_id === own_id ? undefined : nickname(risq, player_id))
    );
  return { title: `${military.length} Military unit${military.length === 1 ? '' : 's'}`, lines };
}
