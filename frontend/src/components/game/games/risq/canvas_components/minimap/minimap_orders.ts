import { drawArrow } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import { addPoint2D, equalsPoint2D, multiplyPoint2D } from '../../../../util/objects2d';
import { ColorRGB } from '../../../../../../scripts/color_rgb';
import { orderArrowColor } from '../../application/orders/orders_model';
import type { RisqBuilding, RisqUnit } from '../../model/types';
import type { RisqOrderProjection } from '../../rendering/board/order_overlays';
import { orderLinesVisible } from '../../rendering/board/order_overlays';
import { unitDotOffsets } from '../../rendering/overview_dots';
import { buildingOrdersVisibleInViewMode, unitVisibleInViewMode } from '../../rendering/terrain';
import { getRisqZone, zoneCenterOffset } from '../../rendering/zones/geometry';
import type { DwgRisq } from '../../risq';

export declare interface MinimapView {
  hexR(): number;
  toCanvas(coordinate: Point2D): Point2D;
  rotation(): number;
}

/** Draws the selected entities' order lines and gather points on the minimap */
export class RisqMinimapOrders {
  constructor(
    private risq: DwgRisq,
    private view: MinimapView
  ) {}

  private unitOffset(unit: RisqUnit): Point2D | undefined {
    if (unit.garrisoned_in !== undefined) {
      return undefined;
    }
    const zone = getRisqZone(this.risq.session.spaceAt(unit.space_coordinate), unit.zone_coordinate);
    return zone
      ? unitDotOffsets(zone, this.risq.getPlayerId(), this.risq.viewport.viewMode()).get(unit.internal_id)
      : undefined;
  }

  drawSelectedOrders(ctx: CanvasRenderingContext2D): void {
    if (!orderLinesVisible(true) || this.risq.session.isReplaying()) {
      return;
    }
    const projection = this.orderProjection();
    ctx.save();
    ctx.lineWidth = 1;
    ctx.setLineDash([3, 2]);
    const view_mode = this.risq.viewport.viewMode();
    for (const id of this.risq.selection.selectedUnitIds()) {
      const unit = this.risq.session.findUnitById(id);
      if (unit && unitVisibleInViewMode(unit.unit_type, view_mode)) {
        this.drawEntityOrders(ctx, unit, projection);
      }
    }
    for (const building of this.risq.selection.selectedBuildings()) {
      if (buildingOrdersVisibleInViewMode(building, view_mode)) {
        this.drawEntityOrders(ctx, building, projection);
      }
      this.drawGatherPoint(ctx, building, projection);
    }
    ctx.restore();
  }

  private drawEntityOrders(
    ctx: CanvasRenderingContext2D,
    entity: RisqUnit | RisqBuilding,
    projection: RisqOrderProjection
  ): void {
    let from = this.entityPoint(entity);
    if (!from) {
      return;
    }
    for (const order of this.risq.orders_model.effectiveForSubject(
      entity.internal_id,
      'unit_id' in entity ? 'unit' : 'building'
    )) {
      const to = this.risq.order_overlays.orderTargetPoint(order, false, from, projection);
      if (!to || equalsPoint2D(from, to)) {
        continue;
      }
      ctx.strokeStyle = orderArrowColor(order.order_type);
      ctx.fillStyle = ctx.strokeStyle;
      drawArrow(ctx, from, to, 3);
      from = to;
    }
  }

  private drawGatherPoint(
    ctx: CanvasRenderingContext2D,
    building: RisqBuilding,
    projection: RisqOrderProjection
  ): void {
    const from = this.entityPoint(building);
    const gather_point = this.risq.commands.gather_points.point(building);
    const to =
      from && gather_point && this.risq.order_overlays.gatherPointTargetPoint(gather_point, false, from, projection);
    if (!from || !to) {
      return;
    }
    if (!equalsPoint2D(from, to)) {
      ctx.strokeStyle = 'white';
      ctx.fillStyle = 'white';
      drawArrow(ctx, from, to, 3);
    }
    ctx.save();
    ctx.translate(to.x, to.y);
    ctx.rotate(-this.view.rotation());
    const flag = this.risq.getPlayerColoredIcon('risq/icons/garrison_flag', new ColorRGB(255, 255, 255));
    ctx.drawImage(flag, 0, -7, 7, 7);
    ctx.restore();
  }

  private orderProjection(): RisqOrderProjection {
    return {
      hexR: (): number => this.view.hexR(),
      coordinateToCanvas: (space: Point2D): Point2D => this.view.toCanvas(space),
      orderPoint: (space: Point2D, offset: Point2D | undefined, zone_view: boolean): Point2D =>
        addPoint2D(this.view.toCanvas(space), zone_view && offset ? offset : { x: 0, y: 0 }),
      unitAnchorOffset: (unit: RisqUnit): Point2D | undefined => {
        const point = this.entityPoint(unit);
        const location = this.risq.session.unitLocation(unit);
        const center = location && this.view.toCanvas(location.space_coordinate);
        return point && center ? { x: point.x - center.x, y: point.y - center.y } : undefined;
      },
    };
  }

  entityPoint(entity: RisqUnit | RisqBuilding): Point2D | undefined {
    const location = 'unit_id' in entity ? this.risq.session.unitLocation(entity) : entity;
    if (!location) {
      return undefined;
    }
    const offset =
      ('unit_id' in entity ? this.unitOffset(entity) : undefined) ?? zoneCenterOffset(location.zone_coordinate, 1);
    return addPoint2D(this.view.toCanvas(location.space_coordinate), multiplyPoint2D(this.view.hexR(), offset));
  }
}
