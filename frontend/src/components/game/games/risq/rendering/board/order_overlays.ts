import { getSettings } from '../../../../../../scripts/settings_store';
import { ColorRGB } from '../../../../../../scripts/color_rgb';
import type { Point2D } from '../../../../util/objects2d';
import { addPoint2D, equalsPoint2D, subtractPoint2D } from '../../../../util/objects2d';
import type { RisqOrdersModel } from '../../application/orders/orders_model';
import { isUnitOrder, orderArrowColor } from '../../application/orders/orders_model';
import { lastAreaAttack } from '../../application/orders/attack_history';
import type { RisqOrderPlanning } from '../../application/orders/planning';
import type { RisqSelection } from '../../application/selection/selection';
import type { RisqSession } from '../../application/session';
import { cantorPair, invertBuildKey, invertPair, invertZoneKey } from '../../model/coordinates';
import type {
  RisqBuilding,
  RisqFrontendOrder,
  RisqGatherPoint,
  RisqMovePathStep,
  RisqUnit,
  RisqTickAction,
  UnitByTypeData,
} from '../../model/types';
import { RisqGatherObjectType, RisqGatherPointLocationKind, RisqOrderType, RisqUnitType } from '../../model/types';
import type { DwgRisq } from '../../risq';
import { DrawRisqSpaceDetail } from '../space';
import { buildingOrdersVisibleInViewMode, unitVisibleInViewMode } from '../terrain';
import { drawUnitTypeCluster } from '../zones/draw';
import {
  hexagonVertices,
  zoneVertices,
  zoneApproachPoint,
  zoneCenterOffset,
  zoneMercenarySlotOffsets,
} from '../zones/geometry';
import { areaBoundary } from './area_boundary';
import { drawSelectionAttackRanges } from './attack_ranges';
import { RisqLinks } from '../links';
import { RisqReplayIntents } from './replay_intents';
import type { RisqViewport } from './viewport';

/** Whether order lines and arrows draw in the given selection pass under the display setting */
export function orderLinesVisible(selected_pass: boolean): boolean {
  const visibility = getSettings().risq_display.order_visibility;
  return visibility === 'all' || (visibility === 'selected_only' && selected_pass);
}

export type RisqOrderProjection = Pick<RisqViewport, 'hexR' | 'coordinateToCanvas' | 'orderPoint' | 'unitAnchorOffset'>;

const GHOST_ALPHA = 0.6;

/** Collapses pending hires into at most `capacity` slots: one per hire, then one per unit type, then one for all */
function mercenaryGhostSlots(player_id: number, unit_ids: number[], capacity: number): UnitByTypeData[][] {
  const group = (unit_id: number, indices: number[]): UnitByTypeData => ({
    player_id,
    unit_id,
    unit_type: RisqUnitType.NONE,
    units: new Set(indices),
  });
  if (unit_ids.length <= capacity) {
    return unit_ids.map((unit_id, i) => [group(unit_id, [i])]);
  }
  const indices_by_type = new Map<number, number[]>();
  unit_ids.forEach((unit_id, i) => indices_by_type.set(unit_id, [...(indices_by_type.get(unit_id) ?? []), i]));
  const by_type = [...indices_by_type.entries()].map(([unit_id, indices]) => group(unit_id, indices));
  return by_type.length <= capacity ? by_type.map((g) => [g]) : [by_type];
}

function drawMercenaryGhosts(
  ctx: CanvasRenderingContext2D,
  risq: DwgRisq,
  slots: UnitByTypeData[][],
  positions: Point2D[],
  unit_r: number,
  rotation: number
): void {
  ctx.globalAlpha = GHOST_ALPHA;
  ctx.textAlign = 'left';
  ctx.textBaseline = 'top';
  for (const [i, slot] of slots.entries()) {
    ctx.save();
    ctx.translate(positions[i].x, positions[i].y);
    ctx.rotate(-rotation);
    const total = slot.reduce((sum, t) => sum + t.units.size, 0);
    drawUnitTypeCluster(ctx, risq, slot, { x: unit_r, y: unit_r }, total, 'white', 'rgba(210, 210, 210, 0.4)', true);
    ctx.restore();
  }
  ctx.globalAlpha = 1;
}

export class RisqOrderOverlays {
  private tinted_attack_areas = new Set<string>();
  private links: RisqLinks;
  private replay_intents: RisqReplayIntents;
  constructor(
    private risq: DwgRisq,
    private session: RisqSession,
    private viewport: RisqViewport,
    private orders_model: RisqOrdersModel,
    private planning: RisqOrderPlanning,
    private selection: RisqSelection
  ) {
    this.links = new RisqLinks(session, viewport);
    this.replay_intents = new RisqReplayIntents(
      session,
      viewport,
      (ctx, from, to, arrow_size) => this.strokeSegment(ctx, from, to, arrow_size),
      (size) => this.cappedSize(size)
    );
  }

  /** Strokes a straight order segment, crossing a link instead when that is the shortest way across the map */
  private strokeSegment(ctx: CanvasRenderingContext2D, from: Point2D, to: Point2D, arrow_size?: number): void {
    this.links.drawLegs(ctx, this.links.legs(from, to), arrow_size);
  }

  private cappedSize(size: number, max_screen_size: number = 3 * size): number {
    const scale = getSettings().risq_display.order_size;
    const scaled_size = size * scale;
    return Math.min(scaled_size, (max_screen_size * scale) / this.viewport.lastTransform().scale);
  }

  draw(ctx: CanvasRenderingContext2D, selected: boolean): void {
    if (this.session.isReplaying()) {
      if (!selected) {
        this.replay_intents.draw(ctx);
      }
      return;
    }
    if (selected) {
      this.drawPendingMercenaries(ctx);
    }
    if (!orderLinesVisible(selected)) {
      return;
    }
    this.tinted_attack_areas.clear();
    this.drawUnitOrders(ctx, selected);
    this.drawBuildingOrders(ctx, selected);
    this.drawGatherPointOrder(ctx, selected);
  }

  drawRanges(ctx: CanvasRenderingContext2D): void {
    drawSelectionAttackRanges(ctx, this.session, this.selection, this.viewport);
  }

  private drawUnitOrders(ctx: CanvasRenderingContext2D, selected_pass: boolean): void {
    const player = this.session.getPlayer();
    if (!player) {
      return;
    }
    const selected = this.selection.selectedUnitIds();
    for (const unit of player.units.values()) {
      if (
        selected.has(unit.internal_id) !== selected_pass ||
        !unitVisibleInViewMode(unit.unit_type, this.viewport.viewMode())
      ) {
        continue;
      }
      this.drawOrdersForUnit(ctx, unit, selected_pass);
    }
  }

  private drawBuildingOrders(ctx: CanvasRenderingContext2D, selected_pass: boolean): void {
    const player = this.session.getPlayer();
    if (!player) {
      return;
    }
    ctx.setLineDash([8, 5]);
    for (const building of player.buildings.values()) {
      if (
        this.selection.isBuildingSelected(building.internal_id) !== selected_pass ||
        !buildingOrdersVisibleInViewMode(building, this.viewport.viewMode())
      ) {
        continue;
      }
      this.drawOrdersForBuilding(ctx, building, selected_pass);
    }
    ctx.setLineDash([]);
    ctx.globalAlpha = 1;
  }

  private drawOrdersForBuilding(ctx: CanvasRenderingContext2D, building: RisqBuilding, selected: boolean): void {
    const zone_view = this.viewport.drawDetail() !== DrawRisqSpaceDetail.OWNERSHIP;
    const building_offset = zoneCenterOffset(building.zone_coordinate, this.viewport.hexR());
    const from = this.viewport.orderPoint(building.space_coordinate, building_offset, true);
    ctx.lineWidth = this.cappedSize(selected ? 2 : 1);
    ctx.globalAlpha = selected ? 1 : 0.35;
    for (const order of this.orders_model.effectiveForSubject(building.internal_id, 'building')) {
      const to = this.orderTargetPoint(order, zone_view, from);
      if (!to || equalsPoint2D(from, to)) {
        continue;
      }
      const color = orderArrowColor(order.order_type);
      ctx.strokeStyle = color;
      ctx.fillStyle = color;
      this.strokeSegment(ctx, from, to, this.cappedSize(selected ? 10 : 6));
    }
  }

  private drawAttackArea(
    ctx: CanvasRenderingContext2D,
    order: RisqFrontendOrder,
    from: Point2D,
    selected: boolean,
    path: RisqMovePathStep[]
  ): Point2D {
    const zone =
      order.order_type === RisqOrderType.OrderType_UnitAttackZone ? invertZoneKey(order.target_id) : undefined;
    const center = this.viewport.coordinateToCanvas(zone?.space ?? invertPair(order.target_id));
    const hex_r = this.viewport.hexR();
    const vertices = (zone ? zoneVertices(zone.zone, hex_r) : hexagonVertices(hex_r)).map(
      (point: Point2D): Point2D => addPoint2D(center, point)
    );
    ctx.save();
    ctx.strokeStyle = orderArrowColor(order.order_type);
    const key = `${order.order_type}:${order.target_id}`;
    if (selected && !this.tinted_attack_areas.has(key)) {
      ctx.beginPath();
      for (const point of vertices) {
        ctx.lineTo(point.x, point.y);
      }
      ctx.closePath();
      ctx.fillStyle = 'rgba(220, 30, 30, 0.18)';
      ctx.fill();
      this.tinted_attack_areas.add(key);
    }
    let current = from;
    for (const step of path) {
      const point = this.viewport.orderPoint(step.space, zoneCenterOffset(step.zone, hex_r), true);
      if (areaBoundary(vertices, point).inside) {
        break;
      }
      if (!equalsPoint2D(current, point)) {
        this.strokeSegment(ctx, current, point);
      }
      current = point;
    }
    const to = areaBoundary(vertices, current).point;
    if (!equalsPoint2D(current, to)) {
      this.strokeSegment(ctx, current, to);
    }
    ctx.restore();
    return to;
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

  private lastAttackPoint(action: RisqTickAction): Point2D | undefined {
    const { target, target_location } = action.execute;
    if (target?.internal_id === undefined || !target_location) {
      return undefined;
    }
    const hex_r = this.viewport.hexR();
    if (target.kind === 'unit') {
      const unit = this.session.findUnitById(target.internal_id);
      const location = unit ? this.session.unitLocation(unit) : undefined;
      if (unit && location) {
        return this.viewport.orderPoint(location.space_coordinate, this.viewport.unitAnchorOffset(unit), true);
      }
      const corpse = this.session.corpseLocation(target.internal_id);
      if (corpse) {
        return this.viewport.orderPoint(
          corpse.space_coordinate,
          this.risq.corpse_layout.worldOffset(corpse.zone, target.internal_id, hex_r),
          true
        );
      }
    } else if (target.kind === 'building') {
      const building = this.session.findBuildingById(target.internal_id);
      if (building) {
        return this.viewport.orderPoint(
          building.space_coordinate,
          zoneCenterOffset(building.zone_coordinate, hex_r),
          true
        );
      }
    }
    return this.viewport.orderPoint(target_location.space, zoneCenterOffset(target_location.zone, hex_r), true);
  }

  private drawOrdersForUnit(ctx: CanvasRenderingContext2D, unit: RisqUnit, selected: boolean): void {
    const location = this.session.unitLocation(unit);
    let orders = this.orders_model.effectiveForSubject(unit.internal_id, 'unit');
    if (
      !orders.length &&
      !unit.active_orders.length &&
      !this.orders_model
        .all()
        .some(
          (order: RisqFrontendOrder): boolean =>
            isUnitOrder(order.order_type) && order.subjects.includes(unit.internal_id)
        )
    ) {
      const attack = lastAreaAttack(unit);
      if (attack?.order) {
        orders = [{ ...attack.order, player_id: unit.player_id, subjects: [unit.internal_id] }];
      }
    }
    if (!location || !orders.length) {
      return;
    }
    const zone_view = this.viewport.drawDetail() !== DrawRisqSpaceDetail.OWNERSHIP;
    let from = this.viewport.orderPoint(location.space_coordinate, this.viewport.unitAnchorOffset(unit), true);
    ctx.lineWidth = this.cappedSize(selected ? 2 : 1);
    ctx.globalAlpha = selected ? 1 : 0.35;
    ctx.setLineDash([8, 5]);
    for (const [i, order] of orders.entries()) {
      const attack = lastAreaAttack(unit, order);
      if (
        !attack &&
        (order.order_type === RisqOrderType.OrderType_UnitAttackSpace ||
          order.order_type === RisqOrderType.OrderType_UnitAttackZone)
      ) {
        const path =
          i === 0 && order.internal_id !== undefined && order.internal_id === unit.active_orders[0]?.internal_id
            ? this.remainingMovePath(unit)
            : [];
        from = this.drawAttackArea(ctx, order, from, selected, path);
        continue;
      }
      if (
        !attack &&
        i === 0 &&
        order.internal_id !== undefined &&
        order.internal_id === unit.active_orders[0]?.internal_id &&
        unit.move_path?.length
      ) {
        from = this.drawMovePath(
          ctx,
          this.remainingMovePath(unit),
          from,
          zone_view,
          selected ? 10 : 6,
          order.order_type
        );
        continue;
      }
      const to = attack ? this.lastAttackPoint(attack) : this.orderTargetPoint(order, zone_view, from);
      if (!to || equalsPoint2D(from, to)) {
        continue;
      }
      const color = orderArrowColor(order.order_type);
      ctx.strokeStyle = color;
      ctx.fillStyle = color;
      this.strokeSegment(ctx, from, to, this.cappedSize(selected ? 10 : 6));
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
    arrow_size: number,
    order_type: RisqOrderType
  ): Point2D {
    const color = orderArrowColor(order_type);
    ctx.strokeStyle = color;
    ctx.fillStyle = color;
    let current = from;
    for (const [i, step] of path.entries()) {
      const final_zone_target =
        zone_view &&
        i === path.length - 1 &&
        (order_type === RisqOrderType.OrderType_UnitMoveZone || order_type === RisqOrderType.OrderType_UnitAttackZone);
      const offset = final_zone_target
        ? zoneApproachPoint(
            step.zone,
            this.viewport.hexR(),
            subtractPoint2D(current, this.viewport.coordinateToCanvas(step.space))
          )
        : zoneCenterOffset(step.zone, this.viewport.hexR());
      const to = this.viewport.orderPoint(step.space, offset, true);
      if (equalsPoint2D(current, to)) {
        continue;
      }
      this.strokeSegment(ctx, current, to, i === path.length - 1 ? this.cappedSize(arrow_size) : undefined);
      current = to;
    }
    return current;
  }

  orderTargetPoint(
    order: RisqFrontendOrder,
    zone_view: boolean,
    from: Point2D,
    projection: RisqOrderProjection = this.viewport
  ): Point2D | undefined {
    if (!this.session.getGame()) {
      return undefined;
    }
    const hex_r = projection.hexR();
    let target_space: Point2D;
    let target_offset: Point2D | undefined;
    switch (order.order_type) {
      case RisqOrderType.OrderType_UnitMoveSpace:
      case RisqOrderType.OrderType_UnitAttackSpace:
        target_space = invertPair(order.target_id);
        break;
      case RisqOrderType.OrderType_UnitMoveZone:
      case RisqOrderType.OrderType_UnitAttackZone: {
        const decoded = invertZoneKey(order.target_id);
        target_space = decoded.space;
        const space_canvas = projection.coordinateToCanvas(target_space);
        target_offset = zone_view
          ? zoneApproachPoint(decoded.zone, hex_r, subtractPoint2D(from, space_canvas))
          : zoneCenterOffset(decoded.zone, hex_r);
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
        target_offset = projection.unitAnchorOffset(target_unit);
        break;
      }
      default:
        return undefined;
    }
    return projection.orderPoint(target_space, target_offset, true);
  }

  gatherPointTargetPoint(
    gather_point: RisqGatherPoint,
    zone_view: boolean,
    from: Point2D,
    projection: RisqOrderProjection = this.viewport
  ): Point2D | undefined {
    if (gather_point.object_type === RisqGatherObjectType.BUILDING && gather_point.object_id === 0) {
      const { space, zone } = invertZoneKey(gather_point.location_id);
      return projection.orderPoint(space, zoneCenterOffset(zone, projection.hexR()), true);
    }
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
    return this.orderTargetPoint({ player_id, order_type, target_id, subjects: [] }, zone_view, from, projection);
  }

  private drawGatherPointOrder(ctx: CanvasRenderingContext2D, selected_pass: boolean): void {
    const visibility = getSettings().risq_display.order_visibility;
    if (visibility === 'all') {
      const player = this.session.getPlayer();
      if (!player) {
        return;
      }
      const selected = new Set(this.selection.selectedBuildings().map((b) => b.internal_id));
      for (const building of player.buildings.values()) {
        if (selected.has(building.internal_id) === selected_pass) {
          this.drawBuildingGatherPoint(ctx, building);
        }
      }
    } else if (selected_pass) {
      for (const building of this.selection.selectedBuildings()) {
        this.drawBuildingGatherPoint(ctx, building);
      }
    }
  }

  private drawBuildingGatherPoint(ctx: CanvasRenderingContext2D, building: RisqBuilding): void {
    const gather_point = this.risq.commands.gather_points.point(building);
    if (!building || !gather_point) {
      return;
    }
    const zone_view = this.viewport.drawDetail() !== DrawRisqSpaceDetail.OWNERSHIP;
    const building_offset = zoneCenterOffset(building.zone_coordinate, this.viewport.hexR());
    const from = this.viewport.orderPoint(building.space_coordinate, building_offset, true);
    const to = this.gatherPointTargetPoint(gather_point, zone_view, from);
    if (!to || equalsPoint2D(from, to)) {
      return;
    }
    ctx.lineWidth = this.cappedSize(2);
    ctx.globalAlpha = 1;
    ctx.setLineDash([8, 5]);
    ctx.strokeStyle = 'white';
    ctx.fillStyle = 'white';
    this.strokeSegment(ctx, from, to, this.cappedSize(10));
    ctx.setLineDash([]);
    const flag_icon = this.risq.getPlayerColoredIcon('risq/icons/garrison_flag', new ColorRGB(255, 255, 255));
    const flag_size = this.cappedSize(Math.max(16, 0.15 * this.viewport.hexR()), 48);
    ctx.drawImage(flag_icon, to.x, to.y - flag_size, flag_size, flag_size);
  }

  private drawPendingMercenaries(ctx: CanvasRenderingContext2D): void {
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
      drawMercenaryGhosts(ctx, this.risq, slots, positions, this.viewport.unitRadius(), rotation);
    }
  }
}
