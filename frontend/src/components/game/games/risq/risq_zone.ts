import {ColorRGB} from "../../../../scripts/color_rgb";
import {drawEllipse} from "../../util/canvas_util";
import {Point2D} from "../../util/objects2d";
import {buildingImage} from "./risq_buildings";
import {RisqZone, UnitByTypeData} from "./risq_data";
import {unitImage} from "./risq_unit";

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
      color.addColor(210, 210, 210, 0.05);
    } else {
      color.addColor(190, 190, 190, 0.03);
    }
  }
  ctx.fillStyle = color.getString();
}

/** Draws the input risq zone */
export function drawRisqZone(ctx: CanvasRenderingContext2D, zone: RisqZone,
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
  if (zone.hovered_data.length !== 3) {
    const r_part = {x: 0.5 * rp, y: 0.5 * rp};
    zone.hovered_data = [
      {c: p1, r: r_part}, {c: p2, r: r_part}, {c: p3, r: r_part},
    ];
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
      case 0: // building
        ctx.drawImage(buildingImage(zone.building), -part.r.x, -part.r.y, 2 * part.r.x, 2 * part.r.y);
        break;
      case 1: // units
        if (!zone.units_by_type) {
          organizeZoneUnits(zone);
        }
        if (zone.units_by_type.size === 0) {
          ctx.strokeStyle = 'rgba(200, 200, 200, 0.5)';
          drawText(ctx, '0', 1.4 * part.r.y, -0.5 * part.r.x, -0.7 * part.r.y, part.r.x);
        } else if (zone.units_by_type.size === 1) {
          const unit_data = [...zone.units_by_type.values()][0];
          const unit = zone.units.get([...unit_data.units.values()][0]);
          ctx.drawImage(unitImage(unit), -part.r.x, -part.r.y, 2 * part.r.x, 2 * part.r.y);
          drawText(ctx, unit_data.units.size.toString(), 1.4 * part.r.y, -part.r.x, -0.7 * part.r.y, 2 * part.r.x);
        } else if (zone.units_by_type.size === 2) {
          const unit_data = [...zone.units_by_type.values()];
          for (let j = 0; j < 2; j++) {
            const units = [...unit_data[j].units.values()];
            const unit = zone.units.get(units[0]);
            ctx.drawImage(unitImage(unit), (0.5 * j - 1) * part.r.x, (0.5 * j - 1) * part.r.y, 1.5 * part.r.x, 1.5 * part.r.y);
            drawText(ctx, units.length.toString(), part.r.y, (0.5 * j - 1) * part.r.x, (0.5 * j - 1) * part.r.y, 2 * part.r.x);
          }
        } else if (zone.units_by_type.size === 3) {
          const unit_data = [...zone.units_by_type.values()];
          for (let j = 0; j < 2; j++) {
            const units = [...unit_data[j].units.values()];
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
          const units = [...unit_data[2].units.values()];
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
        } else if (zone.units_by_type.size === 4) {
          const unit_data = [...zone.units_by_type.values()];
          for (let j = 0; j < 2; j++) {
            const units = [...unit_data[j].units.values()];
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
            const units = [...unit_data[2+j].units.values()];
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
          const unit_data = [...zone.units_by_type.values()];
          for (let j = 0; j < 2; j++) {
            const units = [...unit_data[j].units.values()];
            const unit = zone.units.get(units[0]);
            ctx.drawImage(unitImage(unit), (0.8 * j - 0.9) * part.r.x, -0.9 * part.r.y, part.r.x, part.r.y);
          }
          for (let j = 0; j < 1; j++) {
            const units = [...unit_data[2+j].units.values()];
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
