import {ColorRGB} from "../../../../scripts/color_rgb";
import {atangent} from "../../../../scripts/math";
import {drawEllipse} from "../../util/canvas_util";
import {Point2D, pointInHexagon, rotatePoint, subtractPoint2D} from "../../util/objects2d";
import {DwgRisq} from "./risq";
import {buildingImage} from "./risq_buildings";
import {RisqSpace, RisqZone, UnitByTypeData} from "./risq_data";
import {resourceImage} from "./risq_resources";
import {unitImage} from "./risq_unit";

/** Multiplier for inner zone relative to whole radius */
export const INNER_ZONE_MULTIPLIER = 0.4;

/** Organizes units by unit id for easier processing */
export function organizeZoneUnits(zone: RisqZone) {
  zone.units_by_type = new Map<number, UnitByTypeData>();
  for (const unit of zone.units.values()) {
    if (zone.units_by_type.has(unit.unit_id)) {
      zone.units_by_type.get(unit.unit_id).units.add(unit.internal_id);
    } else {
      zone.units_by_type.set(unit.unit_id, {
        unit_id: unit.unit_id,
        units: new Set<number>([unit.internal_id]),
      });
    }
  }
}

/** Sets zone fill for the input zone */
export function setZoneFill(ctx: CanvasRenderingContext2D, zone: RisqZone) {
  ctx.strokeStyle = 'rgba(250, 250, 250, 0.9)';
  const color = new ColorRGB(10, 120, 10, 0.8);
  if (zone.hovered && !zone.hovered_data.some(p => p.hovered)) {
    if (zone.clicked) {
      color.addColor(210, 210, 210, 0.06);
    } else {
      color.addColor(190, 190, 190, 0.03);
    }
  }
  ctx.fillStyle = color.getString();
}

/** Draws the input risq zone */
export function drawRisqZone(ctx: CanvasRenderingContext2D, game: DwgRisq, zone: RisqZone,
  r: number, rotation: number, p1: Point2D, p2: Point2D, p3: Point2D
) {
  function drawText(ctx: CanvasRenderingContext2D, s: string, ts: number,
    x: number, y: number, w: number, fill_style = 'white')
  {
    const fs = ctx.fillStyle;
    ctx.fillStyle = fill_style;
    ctx.font = `bold ${ts}px serif`;
    ctx.fillText(s, x, y, w);
    ctx.fillStyle = fs;
  }

  ctx.textAlign = 'left';
  ctx.textBaseline = 'top';
  const rp = 0.5 * r;
  if (zone.hovered_data.length !== 3 || zone.reset_hovered_data) {
    const r_part = {x: 0.5 * rp, y: 0.5 * rp};
    zone.hovered_data = [
      {c: p1, r: r_part}, {c: p2, r: r_part}, {c: p3, r: r_part},
    ];
    zone.reset_hovered_data = false;
  } else {
    zone.hovered_data[0].c = p1;
    zone.hovered_data[1].c = p2;
    zone.hovered_data[2].c = p3;
  }
  for (const [i, part] of zone.hovered_data.entries()) {
    ctx.strokeStyle = 'transparent';
    if (part.hovered) {
      if (part.clicked) {
        ctx.fillStyle = 'rgba(200, 200, 200, 0.4)';
      } else {
        ctx.fillStyle = 'rgba(200, 200, 200, 0.2)';
      }
    } else {
      ctx.fillStyle = 'transparent';
    }
    ctx.translate(part.c.x, part.c.y);
    ctx.rotate(-rotation);
    switch(i) {
      case 0: // resources / building
        if (!!zone.resource) {
          ctx.drawImage(game.getIcon(resourceImage(zone.resource)),
            -part.r.x, -part.r.y, 2 * part.r.x, 2 * part.r.y);
        } else {
          ctx.drawImage(game.getIcon(buildingImage(zone.building)),
            -part.r.x, -part.r.y, 2 * part.r.x, 2 * part.r.y);
        }
        break;
      case 1: // economic units
      case 2: // military units
        if (!zone.units_by_type) {
          organizeZoneUnits(zone);
        }
        const units = [...zone.units_by_type.values()];
        const units_by_type = i === 1 ? units.filter(u => u.unit_id < 11) : units.filter(u => u.unit_id > 10);
        if (units_by_type.length === 0) {
          ctx.strokeStyle = 'rgba(200, 200, 200, 0.2)';
        } else if (units_by_type.length === 1) {
          const unit = zone.units.get([...units_by_type[0].units.values()][0]);
          ctx.drawImage(unitImage(unit), -part.r.x, -part.r.y, 2 * part.r.x, 2 * part.r.y);
          drawText(ctx, units_by_type[0].units.size.toString(),
            1.4 * part.r.y, -part.r.x, -0.7 * part.r.y, 2 * part.r.x);
        } else if (units_by_type.length === 2) {
          for (let j = 0; j < 2; j++) {
            const units = [...units_by_type[j].units.values()];
            const unit = zone.units.get(units[0]);
            ctx.drawImage(unitImage(unit), (0.5 * j - 1) * part.r.x, (0.5 * j - 1) * part.r.y, 1.5 * part.r.x, 1.5 * part.r.y);
            drawText(ctx, units.length.toString(), part.r.y, (0.5 * j - 1) * part.r.x, (0.5 * j - 1) * part.r.y, 2 * part.r.x);
          }
        } else if (units_by_type.length === 3) {
          for (let j = 0; j < 2; j++) {
            const units = [...units_by_type[j].units.values()];
            const unit = zone.units.get(units[0]);
            ctx.drawImage(unitImage(unit), (0.8 * j - 0.9) * part.r.x, -0.9 * part.r.y, part.r.x, part.r.y);
            drawText(
              ctx,
              units.length.toString(),
              0.75 * part.r.y,
              (0.8 * j - 1) * part.r.x,
              -0.9 * part.r.y,
              1.5 * part.r.x
            );
          }
          const units = [...units_by_type[2].units.values()];
          const unit = zone.units.get(units[0]);
          ctx.drawImage(unitImage(unit), -0.5 * part.r.x, -0.1 * part.r.y, part.r.x, part.r.y);
          drawText(
            ctx,
            units.length.toString(),
            0.75 * part.r.y,
            -0.5 * part.r.x,
            -0.1 * part.r.y,
            1.5 * part.r.x
          );
        } else if (units_by_type.length === 4) {
          for (let j = 0; j < 2; j++) {
            const units = [...units_by_type[j].units.values()];
            const unit = zone.units.get(units[0]);
            ctx.drawImage(unitImage(unit), (0.8 * j - 0.9) * part.r.x, -0.9 * part.r.y, part.r.x, part.r.y);
            drawText(
              ctx,
              units.length.toString(),
              0.75 * part.r.y,
              (0.8 * j - 0.9) * part.r.x,
              -0.9 * part.r.y,
              1.5 * part.r.x
            );
          }
          for (let j = 0; j < 2; j++) {
            const units = [...units_by_type[2+j].units.values()];
            const unit = zone.units.get(units[0]);
            ctx.drawImage(unitImage(unit), (0.8 * j - 0.9) * part.r.x, -0.1 * part.r.y, part.r.x, part.r.y);
            drawText(
              ctx,
              units.length.toString(),
              0.75 * part.r.y,
              (0.8 * j - 0.9) * part.r.x,
              -0.1 * part.r.y,
              1.5 * part.r.x
            );
          }
        } else {
          for (let j = 0; j < 2; j++) {
            const units = [...units_by_type[j].units.values()];
            const unit = zone.units.get(units[0]);
            ctx.drawImage(unitImage(unit), (0.8 * j - 0.9) * part.r.x, -0.9 * part.r.y, part.r.x, part.r.y);
          }
          for (let j = 0; j < 1; j++) {
            const units = [...units_by_type[2+j].units.values()];
            const unit = zone.units.get(units[0]);
            ctx.drawImage(unitImage(unit), (0.8 * j - 0.9) * part.r.x, -0.1 * part.r.y, part.r.x, part.r.y);
          }
          drawText(
            ctx,
            '...',
            0.8 * part.r.y,
            0.1 * part.r.x,
            -0.1 * part.r.y,
            part.r.x,
            'rgba(150, 150, 150, 0.8)'
          );
          drawText(
            ctx,
            zone.units.size.toString(),
            1.4 * part.r.y,
            -part.r.x,
            -0.7 * part.r.y,
            2 * part.r.x
          );
        }
        break;
      case 2: // resources
        ctx.strokeStyle = 'rgba(200, 200, 200, 0.2)';
        ctx.fillStyle = 'transparent';
        break;
      case 3: // ??
        ctx.strokeStyle = 'rgba(200, 200, 200, 0.2)';
        ctx.fillStyle = 'transparent';
        break;
      default:
        console.error('No implemented');
        break;
    }
    ctx.rotate(rotation);
    ctx.translate(-part.c.x, -part.c.y);
    drawEllipse(ctx, part.c, part.r);
  }
}

/** Resolves hover logic for the zones of a risq space */
export function resolveHoveredZones(p: Point2D, space: RisqSpace, r: number): RisqZone|undefined {
  if (!space.zones) {
    return undefined;
  }

  const resolveZoneDependencies = (m: Point2D, zone: RisqZone, rotate: number) => {
    zone.hovered = true;
    const p = rotatePoint(m, rotate);
    for (const part of zone.hovered_data) {
      const dx = p.x - part.c.x;
      const dy = p.y - part.c.y;
      if ((dx * dx / (part.r.x * part.r.x)) + (dy * dy / (part.r.y * part.r.y)) <= 1) {
        part.hovered = true;
      } else {
        part.hovered = false;
      }
    }
  };

  const m = subtractPoint2D({x: p.x, y: p.y}, space.center);
  let new_hovered_zone: RisqZone|undefined = undefined;
  if (pointInHexagon(m, INNER_ZONE_MULTIPLIER * r)) {
    new_hovered_zone = space.zones[1][1];
    resolveZoneDependencies(m, new_hovered_zone, 0);
  } else {
    const angle = atangent(m.y, m.x);
    let index = Math.floor((angle + Math.PI / 6) / (Math.PI / 3));
    let direction_vector: Point2D = {x: 0, y: 0};
    switch(index) {
      case 6:
        index = 0;
      case 0:
        direction_vector = {x: 1, y: 2};
        break;
      case 1:
        direction_vector = {x: 0, y: 1};
        break;
      case 2:
        direction_vector = {x: 0, y: 0};
        break;
      case 3:
        direction_vector = {x: 1, y: 0};
        break;
      case 4:
        direction_vector = {x: 2, y: 0};
        break;
      case 5:
        direction_vector = {x: 2, y: 1};
        break;
      default:
        console.error('Unknown zone hovered', angle);
        return;
    }
    new_hovered_zone = space.zones[direction_vector.x][direction_vector.y];
    resolveZoneDependencies(m, new_hovered_zone, -(Math.PI / 3) * (1 + 5 - index));
  }
  return new_hovered_zone;
}

/** Removes all hovered flags from the risq zone */
export function unhoverRisqZone(zone: RisqZone) {
  if (!zone) {
    return;
  }
  zone.clicked = false;
  zone.hovered = false;
  for (const part of zone.hovered_data) {
    part.clicked = false;
    part.hovered = false;
  }
}
