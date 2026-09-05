import type { Point2D } from '../../../../util/objects2d';
import { axialDistance, equalsPoint2D } from '../../../../util/objects2d';
import { buildingImage } from '../../risq_buildings';
import { coordinateToIndex, getSpace, invertBuildKey, invertPair, invertZoneKey } from '../../risq_coordinates';
import { RisqOrderType, RisqResourceType } from '../../risq_data';
import { isBuildingOrder, isPlayerOrder, isUnitOrder } from '../../risq_orders';
import { resourceTypeImage } from '../../risq_resources';
import { unitImage } from '../../risq_unit';
import { groupUnitsByType } from '../../risq_zone';
import type { UnitByTypeData } from '../../risq_data';
import type { RisqOrderRowConfig } from './order_row_data';

export interface CostChip {
  icon: string;
  amount: number;
}

export interface ResolvedRow {
  icon: string;
  name: string;
  target: string;
  cost: CostChip[];
  subject_icon?: string;
  subject_units?: UnitByTypeData[];
  progress?: number;
}

function shortLabel(order_type: RisqOrderType): string {
  switch (order_type) {
    case RisqOrderType.OrderType_UnitMoveSpace:
    case RisqOrderType.OrderType_UnitMoveZone:
      return 'Move';
    case RisqOrderType.OrderType_UnitGather:
      return 'Gather';
    case RisqOrderType.OrderType_UnitBuild:
      return 'Build';
    case RisqOrderType.OrderType_BuildingCreate:
      return 'Create';
    case RisqOrderType.OrderType_UnitDelete:
      return 'Delete';
    default:
      return 'Order';
  }
}

/** Maps an order (plus its collapsed siblings) to the display data an order row draws */
export function resolveOrderRow(config: RisqOrderRowConfig): ResolvedRow {
  const order = config.order;
  const game = config.game.getGame();
  const player = game?.players[order.player_id];
  // unit and building internal ids are separate counters and can collide, so gate the lookup by order type
  const subject_unit = isUnitOrder(order.order_type) ? player?.units.get(order.subjects[0]) : undefined;
  const subject_building = isBuildingOrder(order.order_type) ? player?.buildings.get(order.subjects[0]) : undefined;
  const board_size = game?.board_size ?? 0;

  const distance_text = (target_space: Point2D, target_zone?: Point2D): string => {
    if (!subject_unit) {
      return '';
    }
    if (!equalsPoint2D(subject_unit.space_coordinate, target_space)) {
      const d = axialDistance(subject_unit.space_coordinate, target_space);
      return `${d} space${d === 1 ? '' : 's'} away`;
    }
    if (!target_zone) {
      return 'here';
    }
    const d = axialDistance(subject_unit.zone_coordinate, target_zone);
    return d === 0 ? 'here' : `${d} zone${d === 1 ? '' : 's'} away`;
  };

  const resource_cost = (
    cost: { food: number; wood: number; stone: number; gold: number } | undefined,
    multiplier = 1
  ): CostChip[] => {
    if (!cost) {
      return [];
    }
    const chips: CostChip[] = [];
    if (cost.wood) {
      chips.push({ icon: resourceTypeImage(RisqResourceType.WOOD), amount: cost.wood * multiplier });
    }
    if (cost.food) {
      chips.push({ icon: resourceTypeImage(RisqResourceType.FOOD), amount: cost.food * multiplier });
    }
    if (cost.stone) {
      chips.push({ icon: resourceTypeImage(RisqResourceType.STONE), amount: cost.stone * multiplier });
    }
    if (cost.gold) {
      chips.push({ icon: resourceTypeImage(RisqResourceType.GOLD), amount: cost.gold * multiplier });
    }
    return chips;
  };

  const zone_resource_name = (space: Point2D, zone: Point2D): string => {
    if (!game) {
      return 'a resource';
    }
    const found = getSpace(game, coordinateToIndex(board_size, space));
    const zone_data = found?.zones?.flat().find((z) => equalsPoint2D(z.coordinate, zone));
    return zone_data?.resource?.display_name ?? 'a resource';
  };

  let base: { icon: string; name: string; target: string; cost: CostChip[]; progress?: number };
  switch (order.order_type) {
    case RisqOrderType.OrderType_UnitMoveSpace: {
      const target_space = invertPair(order.target_id);
      base = { icon: 'icons/move', name: 'Move', target: distance_text(target_space), cost: [] };
      break;
    }
    case RisqOrderType.OrderType_UnitMoveZone: {
      const { space, zone } = invertZoneKey(order.target_id);
      base = { icon: 'icons/move', name: 'Move', target: distance_text(space, zone), cost: [] };
      break;
    }
    case RisqOrderType.OrderType_UnitGather: {
      const { space, zone } = invertZoneKey(order.target_id);
      base = { icon: 'icons/gather', name: 'Gather', target: zone_resource_name(space, zone), cost: [] };
      break;
    }
    case RisqOrderType.OrderType_UnitBuild: {
      const { building_id, space, zone } = invertBuildKey(order.target_id);
      const producible = subject_unit?.builds.find((p) => p.id === building_id);
      const site = game
        ? getSpace(game, coordinateToIndex(board_size, space))
            ?.zones?.flat()
            .find((z) => equalsPoint2D(z.coordinate, zone))?.building
        : undefined;
      const progress =
        site?.under_construction && site.construction_stamina_total > 0
          ? 1 - site.stamina_remaining / site.construction_stamina_total
          : undefined;
      base = {
        icon: buildingImage(building_id),
        name: `Build ${producible?.display_name ?? 'Building'}`,
        target: distance_text(space, zone),
        cost: resource_cost(producible?.cost),
        progress,
      };
      break;
    }
    case RisqOrderType.OrderType_BuildingCreate: {
      const producible = subject_building?.produces.find((p) => p.id === order.target_id);
      const qty = 1 + (config.collapsed_orders?.length ?? 0);
      const queue_item = subject_building?.production_queue.find((i) => i.order_internal_id === order.internal_id);
      const progress =
        queue_item && producible && producible.stamina_cost > 0
          ? 1 - queue_item.stamina_remaining / producible.stamina_cost
          : undefined;
      base = {
        icon: unitImage(order.target_id),
        name:
          qty > 1
            ? `Create ${producible?.display_name ?? 'Unit'} ×${qty}`
            : `Create ${producible?.display_name ?? 'Unit'}`,
        target: '',
        cost: resource_cost(producible?.cost, qty),
        progress,
      };
      break;
    }
    case RisqOrderType.OrderType_CancelOrder: {
      const cancelled = player?.active_orders.find((o) => o.internal_id === order.target_id);
      base = {
        icon: 'icons/close_gray32',
        name: 'Cancel',
        target: cancelled ? shortLabel(cancelled.order_type) : 'an order',
        cost: [],
      };
      break;
    }
    case RisqOrderType.OrderType_CancelFoundation: {
      const { space, zone } = invertZoneKey(order.target_id);
      base = { icon: 'icons/close_gray32', name: 'Cancel Foundation', target: distance_text(space, zone), cost: [] };
      break;
    }
    case RisqOrderType.OrderType_UnitDelete: {
      base = { icon: 'icons/skull128', name: 'Delete', target: '', cost: [] };
      break;
    }
    default:
      base = { icon: 'icons/fist32', name: shortLabel(order.order_type), target: '', cost: [] };
      break;
  }

  if (isPlayerOrder(order.order_type)) {
    return { ...base, subject_icon: 'icons/person64' };
  }
  if (isUnitOrder(order.order_type) && player) {
    const subject_units = groupUnitsByType(player.units, order.subjects);
    if (subject_units.length > 1) {
      return { ...base, subject_units };
    }
  }
  return {
    ...base,
    subject_icon: subject_unit
      ? unitImage(subject_unit.unit_id)
      : subject_building
        ? buildingImage(subject_building.building_id)
        : undefined,
  };
}
