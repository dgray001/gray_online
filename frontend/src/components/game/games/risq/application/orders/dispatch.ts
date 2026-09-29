import type { ColorRGB } from '../../../../../../scripts/color_rgb';
import { RISQ_MESSAGE_WARNING_COLOR } from '../../canvas_components/message_queue';
import { cantorPair } from '../../model/coordinates';
import type { RisqBuilding, RisqFrontendOrder, RisqGatherPoint } from '../../model/types';
import { RisqGatherObjectType, RisqGatherPointLocationKind, RisqOrderType, RisqUnitType } from '../../model/types';
import { canAffordCost, canHaveGatherPoint } from '../../model/rules';
import type { RisqViewport } from '../../rendering/board/viewport';
import { hoveredZoneObject } from '../../rendering/zones/hit_testing';
import type { RisqCommands } from '../commands';
import type { RisqArmedState } from '../input/armed_state';
import type { RisqHover } from '../input/hover';
import type { RisqSession } from '../session';
import type { RisqOrdersModel } from './orders_model';
import type { RisqOrderPlanning } from './planning';
import type { RisqOrderTargeting } from './targeting';

/** A selected unit as order dispatch needs it */
export declare interface OrderSubject {
  internal_id: number;
  unit_type: RisqUnitType;
}

/** Turns a right-click on the hovered target into orders for the selected units or building */
export class RisqOrderDispatch {
  constructor(
    private session: RisqSession,
    private viewport: RisqViewport,
    private hover: RisqHover,
    private armed: RisqArmedState,
    private targeting: RisqOrderTargeting,
    private planning: RisqOrderPlanning,
    private orders_model: RisqOrdersModel,
    private commands: RisqCommands,
    private show_message: (text: string, color: ColorRGB) => void
  ) {}

  /** Right-click with units selected; a single selected unit may join or resume an existing foundation */
  unitOrder(units: OrderSubject[], single: boolean, ctrl_held: boolean) {
    const space = this.hover.space();
    const zone = this.hover.zone();
    if (!space) {
      this.armed.disarmOrder();
      return;
    }
    const ids = (filter: (u: OrderSubject) => boolean = () => true) => units.filter(filter).map((u) => u.internal_id);
    const economic = (u: OrderSubject) => u.unit_type === RisqUnitType.ECONOMIC;
    const attackers = () => ids((u) => this.armed.getArmedOrder() !== RisqOrderType.NONE || !economic(u));
    const add = (order_type: RisqOrderType, subjects: number[], target_id: number) =>
      this.addUnitOrder(order_type, subjects, target_id, ctrl_held);
    const order_type = this.targeting.resolveActiveOrderType(ctrl_held);
    switch (order_type) {
      case RisqOrderType.OrderType_UnitMoveSpace:
        add(
          order_type,
          ids((u) => !this.targeting.isUnitAtHoverTarget(u.internal_id, false, ctrl_held)),
          space.coordinate_key
        );
        break;
      case RisqOrderType.OrderType_UnitMoveZone:
        if (!zone) {
          return;
        }
        add(
          order_type,
          ids((u) => !this.targeting.isUnitAtHoverTarget(u.internal_id, true, ctrl_held)),
          zone.coordinate_key
        );
        break;
      case RisqOrderType.OrderType_UnitAttackBuilding:
      case RisqOrderType.OrderType_UnitGarrison: {
        const target = zone?.building;
        if (!target) {
          return;
        }
        add(order_type, order_type === RisqOrderType.OrderType_UnitGarrison ? ids() : attackers(), target.internal_id);
        break;
      }
      case RisqOrderType.OrderType_UnitAttackUnit: {
        const target_id = this.targeting.hoveredEnemyUnitId();
        if (target_id === undefined) {
          return;
        }
        add(order_type, attackers(), target_id);
        break;
      }
      case RisqOrderType.OrderType_UnitAttackSpace:
        add(order_type, ids(), space.coordinate_key);
        break;
      case RisqOrderType.OrderType_UnitAttackZone:
        if (!zone) {
          return;
        }
        add(order_type, ids(), zone.coordinate_key);
        break;
      case RisqOrderType.OrderType_UnitGather: {
        if (!zone) {
          return;
        }
        const target = zone.building ?? zone.resource;
        const fitted = this.planning.fitGatherCapacity(target, ids(economic));
        if (fitted.at_capacity) {
          this.show_message(`${target!.display_name} is at worker capacity`, RISQ_MESSAGE_WARNING_COLOR);
        }
        add(order_type, fitted.subjects, zone.coordinate_key);
        break;
      }
      case RisqOrderType.OrderType_UnitBuild:
        if (!zone) {
          return;
        }
        if (single) {
          this.singleBuildOrder(units[0].internal_id, zone.coordinate_key, ctrl_held);
        } else if (this.armed.getArmedBuilding()) {
          this.newFoundation(ids(economic), zone.coordinate_key, ctrl_held);
        } else {
          return;
        }
        break;
      case RisqOrderType.OrderType_UnitRepair:
      case RisqOrderType.OrderType_UnitRenew:
        if (!zone?.building) {
          return;
        }
        add(order_type, ids(economic), zone.building.internal_id);
        break;
      default:
        this.armed.disarmOrder();
        return;
    }
    this.armed.disarmOrder();
  }

  private addUnitOrder(order_type: RisqOrderType, subjects: number[], target_id: number, ctrl_held: boolean) {
    if (subjects.length === 0) {
      return;
    }
    if (order_type === RisqOrderType.OrderType_UnitRenew) {
      const player = this.session.getPlayer();
      const cost = this.session.findBuildingById(target_id)?.renew_cost;
      if (player && cost && !canAffordCost(player, cost)) {
        this.show_message('Not enough resources', RISQ_MESSAGE_WARNING_COLOR);
        return;
      }
    }
    this.orders_model.add({
      player_id: this.session.getPlayerId(),
      order_type,
      subjects,
      target_id,
      clear_previous_orders: !ctrl_held,
    });
  }

  /** Joins this zone's local foundation, or builds the armed building, the unfinished building, or the server foundation */
  private singleBuildOrder(internal_id: number, zone_key: number, ctrl_held: boolean) {
    const local_foundation = this.planning.getLocalFoundation(zone_key);
    if (local_foundation) {
      if (!local_foundation.order.subjects.includes(internal_id)) {
        if (!ctrl_held) {
          this.orders_model.cancelForSubject(internal_id);
        }
        local_foundation.order.subjects.push(internal_id);
        this.orders_model.triggerChange();
      }
      return;
    }
    if (this.armed.getArmedBuilding()) {
      this.newFoundation([internal_id], zone_key, ctrl_held);
      return;
    }
    const building_id =
      this.hover.zone()?.building?.building_id ??
      this.session.getPlayer()?.planned_foundations?.get(zone_key)?.building_id;
    if (building_id !== undefined) {
      this.orders_model.add(this.buildOrder(building_id, [internal_id], zone_key, ctrl_held));
    }
  }

  /** Queues a build order for the armed building and records it as a local foundation */
  private newFoundation(subjects: number[], zone_key: number, ctrl_held: boolean) {
    const building = this.armed.getArmedBuilding()!;
    const order = this.buildOrder(building.id, subjects, zone_key, ctrl_held);
    this.orders_model.add(order);
    this.planning.setLocalFoundation({
      coordinate_key: zone_key,
      building_id: building.id,
      display_name: building.display_name,
      order,
    });
  }

  private buildOrder(building_id: number, subjects: number[], zone_key: number, ctrl_held: boolean): RisqFrontendOrder {
    return {
      player_id: this.session.getPlayerId(),
      order_type: RisqOrderType.OrderType_UnitBuild,
      subjects,
      target_id: cantorPair(building_id, zone_key),
      clear_previous_orders: !ctrl_held,
    };
  }

  /** Right-click with an owned building selected: attack the hovered enemy, else set the gather point there */
  buildingOrder(building: RisqBuilding, ctrl_held: boolean) {
    const target = this.targeting.buildingAttackTarget(building);
    if (target) {
      this.orders_model.add({
        player_id: this.session.getPlayerId(),
        order_type:
          target.kind === 'unit'
            ? RisqOrderType.OrderType_BuildingAttackUnit
            : RisqOrderType.OrderType_BuildingAttackBuilding,
        subjects: [building.internal_id],
        target_id: target.internal_id,
        clear_previous_orders: !ctrl_held,
      });
    } else {
      this.gatherPointOrder(building);
    }
    this.armed.disarmGatherPoint();
    this.armed.disarmBuildingAttack();
  }

  private gatherPointOrder(building: RisqBuilding) {
    if (!canHaveGatherPoint(building)) {
      return;
    }
    const zone = this.hover.zone();
    const space = this.hover.space();
    let point: RisqGatherPoint;
    if (this.viewport.zoneView() && zone) {
      point = {
        location_kind: RisqGatherPointLocationKind.ZONE,
        location_id: zone.coordinate_key,
        object_type: RisqGatherObjectType.NONE,
        object_id: 0,
      };
      const hovered = hoveredZoneObject(zone);
      if (hovered?.kind === 'resource' && zone.resource) {
        point.object_type = RisqGatherObjectType.RESOURCE;
        point.object_id = zone.resource.internal_id;
      } else if (hovered?.kind === 'building' && zone.building) {
        point.object_type = RisqGatherObjectType.BUILDING;
        point.object_id = zone.building.internal_id;
      } else if (hovered?.kind === 'unit') {
        const unit_id = [...hovered.groups[0].units][0];
        if (unit_id !== undefined) {
          point.object_type = RisqGatherObjectType.UNIT;
          point.object_id = unit_id;
        }
      }
    } else if (space) {
      point = {
        location_kind: RisqGatherPointLocationKind.SPACE,
        location_id: space.coordinate_key,
        object_type: RisqGatherObjectType.NONE,
        object_id: 0,
      };
    } else {
      return;
    }
    this.commands.setGatherPoint(building.internal_id, point);
  }

  placeMercenary(keep_armed: boolean) {
    const zone = this.targeting.mercenaryTargetZone();
    const invalid_reason = this.targeting.mercenaryPlacementInvalidReason(zone);
    const mercenary = this.armed.getArmedMercenary();
    if (invalid_reason || !zone || !mercenary) {
      this.show_message(invalid_reason ?? 'No valid location', RISQ_MESSAGE_WARNING_COLOR);
      return;
    }
    this.orders_model.add({
      player_id: this.session.getPlayerId(),
      order_type: RisqOrderType.OrderType_BuyMercenary,
      subjects: [],
      target_id: cantorPair(mercenary.id, zone.coordinate_key),
      clear_previous_orders: false,
    });
    if (!keep_armed) {
      this.armed.disarmOrder();
    }
  }
}
