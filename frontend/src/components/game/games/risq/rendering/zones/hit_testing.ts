import type { Point2D } from '../../../../util/objects2d';
import type { RisqSpace, RisqZone, UnitByTypeData } from '../../model/types';
import { rotatePoint, subtractPoint2D, pointInHexagon } from '../../../../util/objects2d';
import { INNER_ZONE_MULTIPLIER } from './geometry';
import { atangent } from '../../../../../../scripts/math';
import { err } from '../../../../../../scripts/log';
/** Resolves hover logic for the zones of a risq space */
export function resolveHoveredZones(
  p: Point2D,
  space: RisqSpace | undefined,
  r: number,
  override_center?: Point2D,
  ignore_parts?: boolean
): RisqZone | undefined {
  if (!space?.zones) {
    return undefined;
  }

  const resolve_zone_dependencies = (m: Point2D, zone: RisqZone, rotate: number) => {
    zone.hovered = true;
    if (ignore_parts) {
      return;
    }
    const p = rotatePoint(m, rotate);
    for (const part of zone.hovered_data) {
      const dx = p.x - part.c.x;
      const dy = p.y - part.c.y;
      if ((dx * dx) / (part.r.x * part.r.x) + (dy * dy) / (part.r.y * part.r.y) <= 1) {
        part.hovered = true;
      } else {
        part.hovered = false;
      }
    }
  };

  const m = subtractPoint2D({ x: p.x, y: p.y }, override_center ?? space.center);
  let new_hovered_zone: RisqZone | undefined = undefined;
  if (pointInHexagon(m, INNER_ZONE_MULTIPLIER * r)) {
    new_hovered_zone = space.zones[1][1];
    resolve_zone_dependencies(m, new_hovered_zone, 0);
  } else if (pointInHexagon(m, r)) {
    const angle = atangent(m.y, m.x);
    let index = Math.floor((angle + Math.PI / 6) / (Math.PI / 3));
    let direction_vector: Point2D = { x: 0, y: 0 };
    switch (index) {
      case 6:
        index = 0;
      case 0:
        direction_vector = { x: 1, y: 2 };
        break;
      case 1:
        direction_vector = { x: 0, y: 1 };
        break;
      case 2:
        direction_vector = { x: 0, y: 0 };
        break;
      case 3:
        direction_vector = { x: 1, y: 0 };
        break;
      case 4:
        direction_vector = { x: 2, y: 0 };
        break;
      case 5:
        direction_vector = { x: 2, y: 1 };
        break;
      default:
        err('Unknown zone hovered', angle);
        return;
    }
    new_hovered_zone = space.zones[direction_vector.x][direction_vector.y];
    resolve_zone_dependencies(m, new_hovered_zone, -(Math.PI / 3) * (1 + 5 - index));
  }
  return new_hovered_zone;
}

export type HoveredZoneObject =
  | { kind: 'building' }
  | { kind: 'resource' }
  | { kind: 'unit'; groups: UnitByTypeData[] };

/** Returns whatever slot (building, resource, or unit group) is under the cursor in this zone, if any */
export function hoveredZoneObject(zone: RisqZone): HoveredZoneObject | undefined {
  for (const [i, part] of zone.hovered_data.entries()) {
    if (!part.hovered) {
      continue;
    }
    if (i === 0) {
      if (zone.building) {
        return { kind: 'building' };
      }
      if (zone.resource) {
        return { kind: 'resource' };
      }
      return undefined;
    }
    const groups = zone.unit_slots?.[i - 1];
    if (groups) {
      return { kind: 'unit', groups };
    }
  }
  return undefined;
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
