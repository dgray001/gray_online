import { ColorRGB } from '../../../../../../scripts/color_rgb';
import { drawArrow, drawLine } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import { addPoint2D, equalsPoint2D, subtractPoint2D } from '../../../../util/objects2d';
import type { RisqOrdersModel } from '../../application/orders/orders_model';
import { orderArrowColor } from '../../application/orders/orders_model';
import type { RisqOrderPlanning } from '../../application/orders/planning';
import type { RisqSelection } from '../../application/selection/selection';
import type { RisqSession } from '../../application/session';
import { cantorPair, invertBuildKey, invertPair, invertZoneKey } from '../../model/coordinates';
import type { RisqBuilding, RisqFrontendOrder, RisqGatherPoint, RisqMovePathStep, RisqUnit } from '../../model/types';
import { RisqGatherObjectType, RisqGatherPointLocationKind, RisqOrderType } from '../../model/types';
import type { DwgRisq } from '../../risq';
import { unitVisibleInViewMode } from '../zones/draw';
import { zoneApproachPoint, zoneCenterOffset, zoneMercenarySlotOffsets } from '../zones/geometry';
import { drawMercenaryGhosts, mercenaryGhostSlots } from './mercenary';
import type { RisqViewport } from './viewport';

/** Board overlays for the local player's orders: arrows, move paths, gather point, and pending mercenary ghosts */
export class RisqOrderPaths {
  constructor(
    private risq: DwgRisq,
    private session: RisqSession,
    private viewport: RisqViewport,
    private orders_model: RisqOrdersModel,
    private planning: RisqOrderPlanning,
    private selection: RisqSelection
  ) {}

  draw(ctx: CanvasRenderingContext2D) {
    this.drawUnitOrders(ctx);
    this.drawBuildingOrders(ctx);
    this.drawGatherPointOrder(ctx);
    this.drawPendingMercenaries(ctx);
  }

  private drawUnitOrders(ctx: CanvasRenderingContext2D) {
    const player = this.session.getPlayer();
    if (!player || !this.viewport.contentDrawn()) {
      return;
    }
    const selected = this.selection.selectedUnitIds();
    for (const unit of player.units.values()) {
      if (!unitVisibleInViewMode(unit.unit_type, this.viewport.viewMode())) {
        continue;
      }
      this.drawOrdersForUnit(ctx, unit, selected.has(unit.internal_id));
    }
  }

  private drawBuildingOrders(ctx: CanvasRenderingContext2D) {
    const player = this.session.getPlayer();
    if (!player || !this.viewport.contentDrawn()) {
      return;
    }
    ctx.setLineDash([8, 5]);
    for (const building of player.buildings.values()) {
      this.drawOrdersForBuilding(ctx, building, this.selection.isBuildingSelected(building.internal_id));
    }
    ctx.setLineDash([]);
    ctx.globalAlpha = 1;
  }

  private drawOrdersForBuilding(ctx: CanvasRenderingContext2D, building: RisqBuilding, selected: boolean) {
    const zone_view = this.viewport.zoneView();
    const building_offset = zoneCenterOffset(building.zone_coordinate, this.viewport.hexR());
    const from = this.viewport.orderPoint(building.space_coordinate, building_offset, zone_view);
    ctx.lineWidth = selected ? 2 : 1;
    ctx.globalAlpha = selected ? 1 : 0.35;
    for (const order of this.orders_model.effectiveForSubject(building.internal_id, 'building')) {
      const to = this.orderTargetPoint(order, zone_view, from);
      if (!to || equalsPoint2D(from, to)) {
        continue;
      }
      const color = orderArrowColor(order.order_type);
      ctx.strokeStyle = color;
      ctx.fillStyle = color;
      drawArrow(ctx, from, to, selected ? 10 : 6);
    }
  }

  private remainingMovePath(unit: RisqUnit): RisqMovePathStep[] {
    const path = unit.move_path ?? [];
    const location = this.session.unitLocation(unit);
    const current_index = path.findIndex(
      (step) =>
        equalsPoint2D(step.space, location?.space_coordinate) && equalsPoint2D(step.zone, location?.zone_coordinate)
    );
    return path.slice(current_index + 1);
  }

  private drawOrdersForUnit(ctx: CanvasRenderingContext2D, unit: RisqUnit, selected: boolean) {
    const location = this.session.unitLocation(unit);
    const orders = this.orders_model.effectiveForSubject(unit.internal_id, 'unit');
    if (!location || !orders.length) {
      return;
    }
    const zone_view = this.viewport.zoneView();
    let from = this.viewport.orderPoint(location.space_coordinate, this.viewport.unitAnchorOffset(unit), zone_view);
    ctx.lineWidth = selected ? 2 : 1;
    ctx.globalAlpha = selected ? 1 : 0.35;
    ctx.setLineDash([8, 5]);
    for (const [i, order] of orders.entries()) {
      if (i === 0 && order === unit.active_orders[0] && unit.move_path?.length) {
        from = this.drawMovePath(ctx, this.remainingMovePath(unit), from, zone_view, selected ? 10 : 6);
        continue;
      }
      const to = this.orderTargetPoint(order, zone_view, from);
      if (!to || equalsPoint2D(from, to)) {
        continue;
      }
      const color = orderArrowColor(order.order_type);
      ctx.strokeStyle = color;
      ctx.fillStyle = color;
      drawArrow(ctx, from, to, selected ? 10 : 6);
      from = to;
    }
    ctx.setLineDash([]);
    ctx.globalAlpha = 1;
  }

  private drawMovePath(
    ctx: CanvasRenderingContext2D,
    path: RisqMovePathStep[],
    from: Point2D,
    zone_view: boolean,
    arrow_size: number
  ): Point2D {
    const color = orderArrowColor(RisqOrderType.OrderType_UnitMoveZone);
    ctx.strokeStyle = color;
    ctx.fillStyle = color;
    let current = from;
    for (const [i, step] of path.entries()) {
      const to = this.viewport.orderPoint(step.space, zoneCenterOffset(step.zone, this.viewport.hexR()), zone_view);
      if (equalsPoint2D(current, to)) {
        continue;
      }
      if (i === path.length - 1) {
        drawArrow(ctx, current, to, arrow_size);
      } else {
        drawLine(ctx, current, to);
      }
      current = to;
    }
    return current;
  }

  private orderTargetPoint(order: RisqFrontendOrder, zone_view: boolean, from: Point2D): Point2D | undefined {
    if (!this.session.getGame()) {
      return undefined;
    }
    const hex_r = this.viewport.hexR();
    let target_space: Point2D;
    let target_offset: Point2D | undefined;
    switch (order.order_type) {
      case RisqOrderType.OrderType_UnitMoveSpace:
        target_space = invertPair(order.target_id);
        break;
      case RisqOrderType.OrderType_UnitMoveZone:
      case RisqOrderType.OrderType_UnitAttackZone: {
        const decoded = invertZoneKey(order.target_id);
        target_space = decoded.space;
        const space_canvas = this.viewport.coordinateToCanvas(target_space);
        target_offset = zoneApproachPoint(decoded.zone, hex_r, subtractPoint2D(from, space_canvas));
        break;
      }
      case RisqOrderType.OrderType_UnitGather: {
        const decoded = invertZoneKey(order.target_id);
        target_space = decoded.space;
        target_offset = zoneCenterOffset(decoded.zone, hex_r);
        break;
      }
      case RisqOrderType.OrderType_UnitBuild: {
        const decoded = invertBuildKey(order.target_id);
        target_space = decoded.space;
        target_offset = zoneCenterOffset(decoded.zone, hex_r);
        break;
      }
      case RisqOrderType.OrderType_UnitRepair:
      case RisqOrderType.OrderType_UnitRenew:
      case RisqOrderType.OrderType_UnitAttackBuilding:
      case RisqOrderType.OrderType_UnitAutoAttackBuilding:
      case RisqOrderType.OrderType_BuildingAttackBuilding:
      case RisqOrderType.OrderType_BuildingAutoAttackBuilding:
      case RisqOrderType.OrderType_UnitGarrison: {
        const building = this.session.findBuildingById(order.target_id);
        if (!building) {
          return undefined;
        }
        target_space = building.space_coordinate;
        target_offset = zoneCenterOffset(building.zone_coordinate, hex_r);
        break;
      }
      case RisqOrderType.OrderType_UnitAttackUnit:
      case RisqOrderType.OrderType_UnitAutoAttackUnit:
      case RisqOrderType.OrderType_BuildingAttackUnit:
      case RisqOrderType.OrderType_BuildingAutoAttackUnit: {
        const target_unit = this.session.findUnitById(order.target_id);
        const target_location = target_unit ? this.session.unitLocation(target_unit) : undefined;
        if (!target_unit || !target_location) {
          return undefined;
        }
        target_space = target_location.space_coordinate;
        target_offset = this.viewport.unitAnchorOffset(target_unit);
        break;
      }
      default:
        return undefined;
    }
    return this.viewport.orderPoint(target_space, target_offset, zone_view);
  }

  /** Reuses orderTargetPoint's exact placement logic by building the equivalent synthetic order */
  private gatherPointTargetPoint(
    gather_point: RisqGatherPoint,
    zone_view: boolean,
    from: Point2D
  ): Point2D | undefined {
    let order_type: RisqOrderType;
    let target_id: number;
    switch (gather_point.object_type) {
      case RisqGatherObjectType.BUILDING:
        order_type = RisqOrderType.OrderType_UnitAttackBuilding;
        target_id = gather_point.object_id;
        break;
      case RisqGatherObjectType.UNIT:
        order_type = RisqOrderType.OrderType_UnitAttackUnit;
        target_id = gather_point.object_id;
        break;
      case RisqGatherObjectType.RESOURCE:
        order_type = RisqOrderType.OrderType_UnitGather;
        target_id = gather_point.location_id;
        break;
      default:
        order_type =
          gather_point.location_kind === RisqGatherPointLocationKind.SPACE
            ? RisqOrderType.OrderType_UnitMoveSpace
            : RisqOrderType.OrderType_UnitMoveZone;
        target_id = gather_point.location_id;
        break;
    }
    const player_id = this.session.getPlayerId();
    return this.orderTargetPoint({ player_id, order_type, target_id, subjects: [] }, zone_view, from);
  }

  private drawGatherPointOrder(ctx: CanvasRenderingContext2D) {
    const building = this.selection.selectedBuilding();
    const gather_point = building?.gather_point;
    if (!building || !gather_point) {
      return;
    }
    const zone_view = this.viewport.zoneView();
    const building_offset = zoneCenterOffset(building.zone_coordinate, this.viewport.hexR());
    const from = this.viewport.orderPoint(building.space_coordinate, building_offset, zone_view);
    const to = this.gatherPointTargetPoint(gather_point, zone_view, from);
    if (!to || equalsPoint2D(from, to)) {
      return;
    }
    ctx.lineWidth = 2;
    ctx.globalAlpha = 1;
    ctx.setLineDash([8, 5]);
    ctx.strokeStyle = 'white';
    ctx.fillStyle = 'white';
    drawArrow(ctx, from, to, 10);
    ctx.setLineDash([]);
    const flag_icon = this.risq.getPlayerColoredIcon('risq/icons/garrison_flag', new ColorRGB(255, 255, 255));
    const flag_size = Math.max(16, 0.15 * this.viewport.hexR());
    ctx.drawImage(flag_icon, to.x, to.y - flag_size, flag_size, flag_size);
  }

  private drawPendingMercenaries(ctx: CanvasRenderingContext2D) {
    const zone_view = this.viewport.zoneView();
    const hex_r = this.viewport.hexR();
    const groups = new Map<number, { positions: Point2D[]; unit_ids: number[] }>();
    for (const order of this.planning.pendingMercenaryOrders()) {
      const { x: unit_id, y: zone_key } = invertPair(order.target_id);
      const { space, zone } = invertZoneKey(zone_key);
      const key = zone_view ? zone_key : cantorPair(space.x, space.y);
      const space_center = this.viewport.coordinateToCanvas(space);
      const group = groups.get(key) ?? {
        positions: zoneMercenarySlotOffsets(zone_view ? zone : { x: 0, y: 0 }, hex_r).map((offset) =>
          addPoint2D(space_center, offset)
        ),
        unit_ids: [],
      };
      group.unit_ids.push(unit_id);
      groups.set(key, group);
    }
    const rotation = this.viewport.lastTransform().rotation;
    for (const { positions, unit_ids } of groups.values()) {
      const slots = mercenaryGhostSlots(this.session.getPlayerId(), unit_ids, positions.length);
      drawMercenaryGhosts(ctx, this.risq, slots, positions, hex_r, rotation);
    }
  }
}
