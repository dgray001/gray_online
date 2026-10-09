import type { RisqDrawHost } from '../draw_host';
import type { RisqZone } from '../../model/types';
import { RisqVisibilityLevel } from '../../model/types';
import { rotatePoint } from '../../../../util/objects2d';
import { unitCorpseImage } from '../assets/unit';
import { clipToZone, findOuterZoneIndex } from './geometry';

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
