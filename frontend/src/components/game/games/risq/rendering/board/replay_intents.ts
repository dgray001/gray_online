import { drawCircle } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import { equalsPoint2D, subtractPoint2D } from '../../../../util/objects2d';
import type { RisqSession } from '../../application/session';
import type { RisqJournalBuilding, RisqJournalUnit } from '../../model/turn_journal';
import type { RisqTickAction, RisqTickLocation } from '../../model/tick_actions';
import { buildingOrdersVisibleInViewMode, unitVisibleInViewMode } from '../terrain';
import { zoneApproachPoint, zoneCenterOffset } from '../zones/geometry';
import type { RisqViewport } from './viewport';

const INTENT_COLORS: Record<string, string> = {
  move: 'rgba(20, 20, 20, 0.9)',
  gather: 'rgba(255, 150, 0, 0.9)',
  attack: 'rgba(220, 30, 30, 0.9)',
  build: 'rgba(40, 110, 230, 0.9)',
  repair: 'rgba(100, 170, 250, 0.9)',
  renew: 'rgba(190, 200, 40, 0.9)',
  garrison: 'rgba(255, 225, 0, 0.9)',
  ungarrison: 'rgba(255, 225, 0, 0.9)',
  production: 'rgba(160, 90, 220, 0.9)',
  delete: 'rgba(120, 120, 120, 0.9)',
};

const BLOCKED_ALPHA = 0.5;
const RECORD_KINDS = ['order_completed', 'order_cancelled'];

function isDrawnIntent(action: RisqTickAction): boolean {
  const has_target = !!(action.intent.destination ?? action.intent.target_location);
  return (
    !!INTENT_COLORS[action.intent.kind] &&
    !RECORD_KINDS.includes(action.execute.kind) &&
    (has_target || action.intent.kind !== 'move')
  );
}

export type StrokeSegment = (ctx: CanvasRenderingContext2D, from: Point2D, to: Point2D, arrow_size: number) => void;

export class RisqReplayIntents {
  constructor(
    private session: RisqSession,
    private viewport: RisqViewport,
    private stroke_segment: StrokeSegment,
    private capped_size: (size: number) => number
  ) {}

  draw(ctx: CanvasRenderingContext2D): void {
    const tick = this.session.replayIntentTick();
    const journal = this.session.getGame()?.last_turn_journal;
    if (tick === undefined || !journal) {
      return;
    }
    ctx.save();
    ctx.lineWidth = this.capped_size(2);
    for (const unit of journal.units.values()) {
      if (unitVisibleInViewMode(unit.unit_type, this.viewport.viewMode())) {
        this.drawActor(ctx, unit, 'unit', tick);
      }
    }
    for (const building of journal.buildings.values()) {
      const live = this.session.findBuildingById(building.internal_id);
      if (!live || buildingOrdersVisibleInViewMode(live, this.viewport.viewMode())) {
        this.drawActor(ctx, building, 'building', tick);
      }
    }
    ctx.restore();
  }

  private drawActor(
    ctx: CanvasRenderingContext2D,
    actor: RisqJournalUnit | RisqJournalBuilding,
    kind: 'unit' | 'building',
    tick: number
  ): void {
    for (const action of actor.tick_actions.filter((a) => a.tick === tick && isDrawnIntent(a))) {
      const from = this.actorPoint(actor.internal_id, kind, action.intent.location);
      const target = action.intent.destination ?? action.intent.target_location;
      const to = target && this.locationPoint(target, action, from);
      const blocked = action.execute.outcome === 'blocked';
      ctx.globalAlpha = blocked ? BLOCKED_ALPHA : 1;
      ctx.strokeStyle = ctx.fillStyle = INTENT_COLORS[action.intent.kind];
      ctx.setLineDash(blocked ? [3, 4] : [8, 5]);
      if (!to || equalsPoint2D(from, to)) {
        this.drawMarker(ctx, from);
      } else {
        this.stroke_segment(ctx, from, to, this.capped_size(8));
        if (blocked) {
          this.drawBlockedCross(ctx, to);
        }
      }
    }
  }

  private actorPoint(internal_id: number, kind: 'unit' | 'building', location: RisqTickLocation): Point2D {
    const unit = kind === 'unit' ? this.session.findUnitById(internal_id) : undefined;
    const at_location =
      unit?.garrisoned_in === undefined &&
      equalsPoint2D(unit?.space_coordinate, location.space) &&
      equalsPoint2D(unit?.zone_coordinate, location.zone);
    const offset = at_location ? this.viewport.unitAnchorOffset(unit!) : this.zoneOffset(location);
    return this.viewport.orderPoint(location.space, offset, this.viewport.contentDrawn());
  }

  private locationPoint(location: RisqTickLocation, action: RisqTickAction, from: Point2D): Point2D {
    const target = action.intent.target;
    const unit = target?.kind === 'unit' ? this.session.findUnitById(target.internal_id) : undefined;
    const tracks_unit = unit && equalsPoint2D(unit.space_coordinate, location.space);
    const is_destination = location === action.intent.destination;
    const offset = tracks_unit
      ? this.viewport.unitAnchorOffset(unit)
      : is_destination
        ? this.approachOffset(location, from)
        : this.zoneOffset(location);
    return this.viewport.orderPoint(location.space, offset, this.viewport.contentDrawn());
  }

  private approachOffset(location: RisqTickLocation, from: Point2D): Point2D {
    const space_canvas = this.viewport.coordinateToCanvas(location.space);
    return zoneApproachPoint(location.zone, this.viewport.hexR(), subtractPoint2D(from, space_canvas));
  }

  private zoneOffset(location: RisqTickLocation): Point2D {
    return zoneCenterOffset(location.zone, this.viewport.hexR());
  }

  private drawMarker(ctx: CanvasRenderingContext2D, at: Point2D): void {
    ctx.fillStyle = 'transparent';
    drawCircle(ctx, at, 0.2 * this.viewport.hexR());
  }

  private drawBlockedCross(ctx: CanvasRenderingContext2D, at: Point2D): void {
    const half = this.capped_size(4);
    ctx.setLineDash([]);
    ctx.beginPath();
    ctx.moveTo(at.x - half, at.y - half);
    ctx.lineTo(at.x + half, at.y + half);
    ctx.moveTo(at.x + half, at.y - half);
    ctx.lineTo(at.x - half, at.y + half);
    ctx.stroke();
  }
}
