import type { BoardTransformData } from '../../../../../util/canvas_board/canvas_board';
import { createTooltipState, drawTooltip, shouldShowTooltip } from '../../../../../util/canvas_components/tooltip';
import { drawHexagon } from '../../../../../util/canvas_util';
import type { Point2D } from '../../../../../util/objects2d';
import { equalsPoint2D } from '../../../../../util/objects2d';
import type { RectHoverData, RisqSpace, RisqZone } from '../../../model/types';
import { RisqVisibilityLevel } from '../../../model/types';
import { buildingImage, rubbleImage } from '../../../rendering/assets/buildings';
import { resourceIcon } from '../../../rendering/assets/resources';
import { borderStrokeStyle, drawHexImage } from '../../../rendering/space';
import { RisqViewMode, spaceOwnerColor, terrainImage } from '../../../rendering/terrain';
import { getZoneFill } from '../../../rendering/zones/draw';
import { INNER_ZONE_MULTIPLIER, OUTER_ZONE_INDICES } from '../../../rendering/zones/geometry';
import { resolveHoveredZones, unhoverRisqZone } from '../../../rendering/zones/hit_testing';
import type { DwgRisq } from '../../../risq';
import { rectHovered } from './stats_view';
import type { PanelDrawContext } from './primitives';

/** The zoomed hex preview shown for a space or zone, with its hoverable zones and visibility badge */
export class RisqSpaceHexagon {
  private r = 0;
  private c: Point2D = { x: 0, y: 0 };
  private eye_badge_hover: RectHoverData = { ps: { x: 0, y: 0 }, pe: { x: 0, y: 0 } };
  private eye_badge_tooltip = createTooltipState();
  private hovered_zone?: RisqZone;

  hoveredZone(): RisqZone | undefined {
    return this.hovered_zone;
  }

  clearHoveredZone(): void {
    if (!!this.hovered_zone) {
      unhoverRisqZone(this.hovered_zone);
      this.hovered_zone = undefined;
    }
  }

  /** Draws the hex below yi, highlighting curr_zone (given in zone index coordinates); returns the height used */
  draw(
    pc: PanelDrawContext,
    space: RisqSpace,
    separator_distance: number,
    yi: number,
    curr_zone: Point2D = { x: -1, y: -1 }
  ): number {
    const { ctx, frame, risq } = pc;
    const hexagon_height = Math.min(frame.w(), frame.yi() + 0.4 * frame.h() - yi - separator_distance);
    const owner_color = spaceOwnerColor(space.ownership, risq.getGame()?.players ?? []);
    ctx.strokeStyle = borderStrokeStyle(owner_color, 1);
    ctx.lineWidth = 2;
    const r = 0.5 * hexagon_height;
    this.r = r;
    const inner_r = INNER_ZONE_MULTIPLIER * r;
    const c = { x: frame.xc(), y: yi + r };
    this.c = c;
    drawHexImage(ctx, risq.getIcon(terrainImage(space.terrain_id)), c, r);
    ctx.fillStyle = 'transparent';
    drawHexagon(ctx, c, r);
    this.drawEyeBadge(pc, space, yi, hexagon_height);
    if (pc.visibility < RisqVisibilityLevel.FOG || !space.zones) {
      return hexagon_height + separator_distance;
    }
    ctx.strokeStyle = borderStrokeStyle(owner_color, 0.7);
    ctx.lineWidth = 0.5;
    let zone = space.zones[1][1];
    const zone_fill = getZoneFill(zone, RisqViewMode.ALL, undefined, true, 4);
    if (curr_zone.x === 1 && curr_zone.y === 1) {
      zone_fill.addColor(255, 255, 255, 0.2);
    }
    ctx.fillStyle = zone_fill.getString();
    drawHexagon(ctx, c, inner_r);
    this.drawZoneIcon(ctx, risq, zone, c, 0.4 * inner_r);
    const a = Math.PI / 3;
    const mid_r = 0.5 * (inner_r + r);
    const icon_r = 0.32 * (r - inner_r);
    for (let i = 0; i < 6; i++) {
      const direction_vector = OUTER_ZONE_INDICES[i];
      zone = space.zones[direction_vector.x][direction_vector.y];
      const outer_fill = getZoneFill(zone, RisqViewMode.ALL, undefined, true, 4);
      if (curr_zone.x === direction_vector.x && curr_zone.y === direction_vector.y) {
        outer_fill.addColor(255, 255, 255, 0.2);
      }
      ctx.fillStyle = outer_fill.getString();
      ctx.beginPath();
      ctx.lineTo(c.x + inner_r * Math.cos(a * i + Math.PI / 6), c.y + inner_r * Math.sin(a * i + Math.PI / 6));
      ctx.lineTo(c.x + inner_r * Math.cos(a * i + Math.PI / 2), c.y + inner_r * Math.sin(a * i + Math.PI / 2));
      ctx.lineTo(c.x + r * Math.cos(a * i + Math.PI / 2), c.y + r * Math.sin(a * i + Math.PI / 2));
      ctx.lineTo(c.x + r * Math.cos(a * i + Math.PI / 6), c.y + r * Math.sin(a * i + Math.PI / 6));
      ctx.closePath();
      ctx.stroke();
      ctx.fill();
      const angle_mid = a * i + Math.PI / 3;
      this.drawZoneIcon(
        ctx,
        risq,
        zone,
        { x: c.x + mid_r * Math.cos(angle_mid), y: c.y + mid_r * Math.sin(angle_mid) },
        icon_r
      );
    }
    return hexagon_height + separator_distance;
  }

  private drawEyeBadge(pc: PanelDrawContext, space: RisqSpace, yi: number, hexagon_height: number): void {
    if (space.visibility !== RisqVisibilityLevel.SPY && space.visibility !== RisqVisibilityLevel.POOR) {
      this.eye_badge_hover.hovered = false;
      return;
    }
    const { frame } = pc;
    const badge_size = 0.15 * hexagon_height;
    const badge_p =
      space.visibility === RisqVisibilityLevel.POOR
        ? { x: frame.xi() + 0.05 * frame.w(), y: yi }
        : { x: frame.xi() + 0.1 * frame.w(), y: yi + 0.05 * hexagon_height };
    pc.ctx.drawImage(
      pc.risq.getIcon(`icons/${space.visibility === RisqVisibilityLevel.SPY ? 'eye' : 'no_eye'}64`),
      badge_p.x,
      badge_p.y,
      badge_size,
      badge_size
    );
    this.eye_badge_hover.ps = badge_p;
    this.eye_badge_hover.pe = { x: badge_p.x + badge_size, y: badge_p.y + badge_size };
  }

  private drawZoneIcon(ctx: CanvasRenderingContext2D, risq: DwgRisq, zone: RisqZone, p: Point2D, icon_r: number): void {
    let icon: HTMLImageElement | HTMLCanvasElement | undefined;
    if (zone.resource) {
      icon = resourceIcon(risq, zone.resource);
    } else if (zone.building) {
      const building_color = risq.getGame()?.players[zone.building.player_id]?.color;
      const building_image = buildingImage(
        zone.building.building_id,
        zone.building.under_construction,
        false,
        zone.building.combat_stats
      );
      icon = building_color ? risq.getPlayerColoredIcon(building_image, building_color) : risq.getIcon(building_image);
    } else if (zone.destroyed_building) {
      icon = risq.getIcon(rubbleImage(zone.destroyed_building, zone.destroyed_building_turns ?? 1));
    }
    if (icon) {
      ctx.drawImage(icon, p.x - icon_r, p.y - icon_r, 2 * icon_r, 2 * icon_r);
    }
  }

  drawTooltip(
    ctx: CanvasRenderingContext2D,
    transform: BoardTransformData,
    risq: DwgRisq,
    dt: number,
    space: RisqSpace
  ): void {
    if (
      !shouldShowTooltip(this.eye_badge_tooltip, !!this.eye_badge_hover.hovered, !!this.eye_badge_hover.clicked, dt)
    ) {
      return;
    }
    drawTooltip(
      this.eye_badge_tooltip,
      ctx,
      transform,
      risq.viewport.canvasSize(),
      space.visibility === RisqVisibilityLevel.POOR
        ? 'Poor visibility on this space; overall unit count seen but no specific units or zone-level unit information'
        : ''
    );
  }

  mousemove(m: Point2D, space: RisqSpace): void {
    rectHovered(m, this.eye_badge_hover);
    const new_hovered_zone = resolveHoveredZones(m, space, this.r, this.c, true);
    if (!!this.hovered_zone && !equalsPoint2D(this.hovered_zone.coordinate, new_hovered_zone?.coordinate)) {
      this.hovered_zone.hovered = false;
      this.hovered_zone.clicked = false;
    }
    this.hovered_zone = new_hovered_zone;
  }
}
