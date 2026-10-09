import type { Point2D } from '../../../../util/objects2d';
import { rotatePoint } from '../../../../util/objects2d';
import type { RisqCorpse, RisqZone } from '../../model/types';
import { RisqVisibilityLevel } from '../../model/types';
import type { RisqDrawHost } from '../draw_host';
import { unitCorpseImage } from '../assets/unit';
import { seededRandom } from '../../../../../../scripts/math';
import {
  corpseUnitRadius,
  findOuterZoneIndex,
  MAX_UNIT_RINGS,
  zoneCorpsePoint,
  zoneCorpseSlots,
  clipToZone,
} from './geometry';

const CORPSE_JITTER_RADIUS_FRACTION = 0.15;

interface CorpseZoneLayout {
  seed: number;
  corpses?: RisqCorpse[];
  placements: Map<number, Point2D>;
  slots: Map<number, number>;
}

export class RisqCorpseLayout {
  private zones = new Map<number, CorpseZoneLayout>();
  private base_slots = new Map<boolean, Point2D[]>();

  positions(zone: RisqZone): ReadonlyMap<number, Point2D> {
    let layout = this.zones.get(zone.coordinate_key);
    if (!layout) {
      layout = { seed: seededRandom(zone.coordinate_key)(), placements: new Map(), slots: new Map() };
      this.zones.set(zone.coordinate_key, layout);
    }
    if (layout.corpses !== zone.corpses) {
      const active = new Set(zone.corpses.map((corpse: RisqCorpse): number => corpse.internal_id));
      for (const [slot, id] of layout.slots) {
        if (!active.has(id)) {
          layout.slots.delete(slot);
        }
      }
      for (const corpse of zone.corpses) {
        if (!layout.placements.has(corpse.internal_id)) {
          this.assign(zone, layout, corpse);
        }
      }
      layout.corpses = zone.corpses;
    }
    return layout.placements;
  }

  reset(): void {
    this.zones.clear();
  }

  private spots(zone: RisqZone): Point2D[] {
    const center = findOuterZoneIndex(zone.coordinate) === -1;
    let spots = this.base_slots.get(center);
    if (!spots) {
      spots = zoneCorpseSlots(zone.coordinate);
      this.base_slots.set(center, spots);
    }
    return spots;
  }

  private assign(zone: RisqZone, layout: CorpseZoneLayout, corpse: RisqCorpse): void {
    const random = seededRandom(corpse.internal_id);
    const spots = this.spots(zone);
    const start = Math.floor(((random() + layout.seed) % 1) * spots.length);
    let point: Point2D | undefined;
    for (let step = 0; step < spots.length; step++) {
      const slot = (start + step) % spots.length;
      if (!layout.slots.has(slot)) {
        layout.slots.set(slot, corpse.internal_id);
        point = spots[slot];
        break;
      }
    }
    point ??= zoneCorpsePoint(zone.coordinate, Math.floor(((random() + layout.seed) % 1) * MAX_UNIT_RINGS), random());
    const jitter = corpseUnitRadius(1) * CORPSE_JITTER_RADIUS_FRACTION;
    const perturbation = (): number => (random() - 0.5) * 2 * jitter;
    layout.placements.set(corpse.internal_id, { x: point.x + perturbation(), y: point.y + perturbation() });
  }
}

export function drawZoneCorpses(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  zone: RisqZone,
  visibility: number,
  hex_r: number,
  map_rotation: number
): void {
  if (visibility < RisqVisibilityLevel.GOOD || zone.corpses.length === 0) {
    return;
  }
  const direction = findOuterZoneIndex(zone.coordinate);
  const zone_rotation = direction === -1 ? 0 : (Math.PI / 3) * (direction + 1);
  const radius = game.viewport.corpseRadius();
  const positions = game.corpse_layout.positions(zone);
  ctx.save();
  clipToZone(ctx, { x: 0, y: 0 }, hex_r, zone.coordinate);
  for (const corpse of zone.corpses) {
    const point = positions.get(corpse.internal_id)!;
    const p = rotatePoint({ x: point.x * hex_r, y: point.y * hex_r }, zone_rotation);
    const image = unitCorpseImage(corpse.unit_id, corpse.turns);
    const color = game.getGame()?.players[corpse.player_id]?.color;
    const icon = color ? game.getPlayerColoredIcon(image, color) : game.getIcon(image);
    ctx.translate(p.x, p.y);
    ctx.rotate(-map_rotation);
    ctx.drawImage(icon, -radius, -radius, 2 * radius, 2 * radius);
    ctx.rotate(map_rotation);
    ctx.translate(-p.x, -p.y);
  }
  ctx.restore();
}
