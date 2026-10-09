import type { Point2D } from '../../../../util/objects2d';
import { axialDistance, equalsPoint2D } from '../../../../util/objects2d';
import { buildingImage, techImage } from '../../rendering/assets/buildings';
import { coordinateToIndex, getSpace, invertBuildKey, invertPair, invertZoneKey } from '../../model/coordinates';
import { RisqOrderType, RisqProducibleKind, RisqResourceType } from '../../model/types';
import { isBuildingOrder, isPlayerOrder, isUnitOrder } from '../../application/orders/orders_model';
import { resourceTypeImage } from '../../rendering/assets/resources';
import { unitImage } from '../../rendering/assets/unit';
import { groupUnitsByType } from '../../model/unit_groups';
import type { UnitByTypeData } from '../../model/types';
import type { RisqOrderRowConfig } from './order_row';

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
  progress_text?: string;
}

function progressText(total: number | undefined, remaining: number | undefined): string | undefined {
  return total === undefined ? undefined : `Progress: ${total - (remaining ?? total)} / ${total}`;
}

export function shortLabel(order_type: RisqOrderType): string {
  switch (order_type) {
    case RisqOrderType.OrderType_UnitMoveSpace:
    case RisqOrderType.OrderType_UnitMoveZone:
      return 'Move';
    case RisqOrderType.OrderType_UnitGather:
      return 'Gather';
    case RisqOrderType.OrderType_UnitBuild:
      return 'Build';
    case RisqOrderType.OrderType_UnitRepair:
      return 'Repair';
    case RisqOrderType.OrderType_UnitRenew:
      return 'Renew';
    case RisqOrderType.OrderType_BuildingCreate:
      return 'Create';
    case RisqOrderType.OrderType_BuildingResearch:
      return 'Research';
    case RisqOrderType.OrderType_UnitAttackUnit:
    case RisqOrderType.OrderType_UnitAttackBuilding:
    case RisqOrderType.OrderType_UnitAttackZone:
    case RisqOrderType.OrderType_UnitAttackSpace:
    case RisqOrderType.OrderType_BuildingAttackUnit:
    case RisqOrderType.OrderType_BuildingAttackBuilding:
      return 'Attack';
    case RisqOrderType.OrderType_UnitGarrison:
      return 'Garrison';
    case RisqOrderType.OrderType_UnitUngarrison:
      return 'Ungarrison';
    case RisqOrderType.OrderType_UnitDelete:
    case RisqOrderType.OrderType_BuildingDelete:
      return 'Delete';
    case RisqOrderType.OrderType_BuyMercenary:
      return 'Hire';
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
  const subject_position = subject_unit ? config.game.session.unitLocation(subject_unit) : undefined;

  const distance_text = (target_space: Point2D, target_zone?: Point2D): string => {
    if (!subject_position) {
      return '';
    }
    if (!equalsPoint2D(subject_position.space_coordinate, target_space)) {
      const d = axialDistance(subject_position.space_coordinate, target_space);
      return `${d} space${d === 1 ? '' : 's'} away`;
    }
    if (!target_zone) {
      return '';
    }
    const d = axialDistance(subject_position.zone_coordinate, target_zone);
    return d === 0 ? '' : `${d} zone${d === 1 ? '' : 's'} away`;
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
    return zone_data?.resource?.display_name ?? zone_data?.building?.display_name ?? 'a resource';
  };

  let base: ResolvedRow;
  switch (order.order_type) {
    case RisqOrderType.OrderType_UnitMoveSpace: {
      const target_space = invertPair(order.target_id);
      base = { icon: 'icons/move32', name: 'Move', target: distance_text(target_space), cost: [] };
      break;
    }
    case RisqOrderType.OrderType_UnitMoveZone: {
      const { space, zone } = invertZoneKey(order.target_id);
      base = { icon: 'icons/move32', name: 'Move', target: distance_text(space, zone), cost: [] };
      break;
    }
    case RisqOrderType.OrderType_UnitGather: {
      const { space, zone } = invertZoneKey(order.target_id);
      base = { icon: 'icons/gather32', name: 'Gather', target: zone_resource_name(space, zone), cost: [] };
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
        icon: buildingImage(building_id, false, true),
        name: `Build ${producible?.display_name ?? 'Building'}`,
        target: distance_text(space, zone),
        cost: resource_cost(producible?.cost),
        progress,
        progress_text: progressText(
          site?.construction_stamina_total || producible?.stamina_cost,
          site?.stamina_remaining
        ),
      };
      break;
    }
    case RisqOrderType.OrderType_BuildingCreate: {
      const producible = subject_building?.produces.find(
        (p) => p.kind === RisqProducibleKind.UNIT && p.id === order.target_id
      );
      const qty = 1 + (config.collapsed_orders?.length ?? 0);
      const total = producible ? producible.stamina_cost * qty : undefined;
      const remaining = [order, ...(config.collapsed_orders ?? [])].reduce(
        (sum: number, queued: typeof order): number =>
          sum +
          (subject_building?.production_queue.find((i) => i.order_internal_id === queued.internal_id)
            ?.stamina_remaining ??
            producible?.stamina_cost ??
            0),
        0
      );
      const queue_item = subject_building?.production_queue.find((i) => i.order_internal_id === order.internal_id);
      const progress =
        queue_item && producible && producible.stamina_cost > 0
          ? 1 - queue_item.stamina_remaining / producible.stamina_cost
          : undefined;
      base = {
        icon: unitImage(order.target_id, true),
        name:
          qty > 1
            ? `Create ${producible?.display_name ?? 'Unit'} ×${qty}`
            : `Create ${producible?.display_name ?? 'Unit'}`,
        target: '',
        cost: resource_cost(producible?.cost, qty),
        progress,
        progress_text: progressText(total, remaining),
      };
      break;
    }
    case RisqOrderType.OrderType_BuildingResearch: {
      const producible = subject_building?.produces.find(
        (p) => p.kind === RisqProducibleKind.TECH && p.id === order.target_id
      );
      const queue_item = subject_building?.production_queue.find((i) => i.order_internal_id === order.internal_id);
      const progress =
        queue_item && producible && producible.stamina_cost > 0
          ? 1 - queue_item.stamina_remaining / producible.stamina_cost
          : undefined;
      base = {
        icon: techImage(order.target_id),
        name: `Research ${producible?.display_name ?? 'Tech'}`,
        target: '',
        cost: resource_cost(producible?.cost),
        progress,
        progress_text: progressText(producible?.stamina_cost, queue_item?.stamina_remaining),
      };
      break;
    }
    case RisqOrderType.OrderType_UnitAttackUnit:
    case RisqOrderType.OrderType_BuildingAttackUnit: {
      const target_unit = game?.players
        .flatMap((p) => [...p.units.values()])
        .find((u) => u.internal_id === order.target_id);
      const target_location = target_unit ? config.game.session.unitLocation(target_unit) : undefined;
      base = {
        icon: 'icons/sword32',
        name: `Attack ${target_unit?.display_name ?? 'Unit'}`,
        target: target_location ? distance_text(target_location.space_coordinate, target_location.zone_coordinate) : '',
        cost: [],
      };
      break;
    }
    case RisqOrderType.OrderType_UnitAttackBuilding:
    case RisqOrderType.OrderType_BuildingAttackBuilding: {
      const target_building = game?.players
        .flatMap((p) => [...p.buildings.values()])
        .find((b) => b.internal_id === order.target_id);
      base = {
        icon: 'icons/sword32',
        name: `Attack ${target_building?.display_name ?? 'Building'}`,
        target: target_building ? distance_text(target_building.space_coordinate, target_building.zone_coordinate) : '',
        cost: [],
      };
      break;
    }
    case RisqOrderType.OrderType_UnitAttackZone: {
      const { space, zone } = invertZoneKey(order.target_id);
      base = { icon: 'icons/sword32', name: 'Attack Zone', target: distance_text(space, zone), cost: [] };
      break;
    }
    case RisqOrderType.OrderType_UnitAttackSpace: {
      const target_space = invertPair(order.target_id);
      base = { icon: 'icons/sword32', name: 'Attack Space', target: distance_text(target_space), cost: [] };
      break;
    }
    case RisqOrderType.OrderType_UnitGarrison: {
      const building = game?.players
        .flatMap((p) => [...p.buildings.values()])
        .find((b) => b.internal_id === order.target_id);
      base = {
        icon: 'icons/garrison32',
        name: `Garrison in ${building?.display_name ?? 'Building'}`,
        target: building ? distance_text(building.space_coordinate, building.zone_coordinate) : '',
        cost: [],
      };
      break;
    }
    case RisqOrderType.OrderType_UnitUngarrison: {
      base = { icon: 'icons/ungarrison32', name: 'Ungarrison', target: '', cost: [] };
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
    case RisqOrderType.OrderType_BuyMercenary: {
      const { x: unit_id, y: zone_key } = invertPair(order.target_id);
      const { space } = invertZoneKey(zone_key);
      const mercenary = player?.available_mercenaries.find((m) => m.id === unit_id);
      base = {
        icon: unitImage(unit_id, true),
        name: `Hire ${mercenary?.display_name ?? 'Mercenary'}`,
        target: `at (${space.x}, ${space.y})`,
        cost: resource_cost(mercenary?.cost),
      };
      break;
    }
    case RisqOrderType.OrderType_UnitRepair: {
      const building = game?.players
        .flatMap((p) => [...p.buildings.values()])
        .find((b) => b.internal_id === order.target_id);
      base = {
        icon: 'icons/repair32',
        name: `Repair ${building?.display_name ?? 'Building'}`,
        target: building ? distance_text(building.space_coordinate, building.zone_coordinate) : '',
        cost: [],
      };
      break;
    }
    case RisqOrderType.OrderType_UnitRenew: {
      const building = game?.players
        .flatMap((p) => [...p.buildings.values()])
        .find((b) => b.internal_id === order.target_id);
      const resources = building?.renewing ? (building.resources_left ?? 0) : 0;
      const completed =
        building?.renew_stamina && building.resource_capacity
          ? (resources / building.resource_capacity) * building.renew_stamina
          : undefined;
      base = {
        icon: 'icons/wheat32',
        name: `Renew ${building?.display_name ?? 'Building'}`,
        target: building ? distance_text(building.space_coordinate, building.zone_coordinate) : '',
        cost: [],
        progress_text:
          completed === undefined ? undefined : `Progress: ${completed.toFixed(1)} / ${building?.renew_stamina}`,
      };
      break;
    }
    case RisqOrderType.OrderType_UnitDelete: {
      base = { icon: 'icons/skull32', name: 'Delete', target: '', cost: [] };
      break;
    }
    case RisqOrderType.OrderType_BuildingDelete: {
      base = {
        icon: 'icons/skull32',
        name: `Delete ${subject_building?.display_name ?? 'Building'}`,
        target: '',
        cost: [],
      };
      break;
    }
    default:
      base = { icon: 'icons/fist32', name: shortLabel(order.order_type), target: '', cost: [] };
      break;
  }

  if (isPlayerOrder(order.order_type)) {
    return { ...base, subject_icon: 'icons/person32' };
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
      ? unitImage(subject_unit.unit_id, true)
      : subject_building
        ? buildingImage(subject_building.building_id, false, true)
        : undefined,
  };
}
