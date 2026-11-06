import { AXIAL_DIRECTION_VECTORS } from '../../../../util/objects2d';
import type { RisqTickLocation } from '../../model/tick_actions';
import type { RisqSpaceLink } from '../../model/types';

export function zoneKey(location: RisqTickLocation): string {
  return `zone:${location.space.x},${location.space.y},${location.zone.x},${location.zone.y}`;
}

export function seamNeighbors(links: RisqSpaceLink[]): Map<string, RisqTickLocation> {
  const neighbors = new Map<string, RisqTickLocation>();
  for (const link of links) {
    const from = { space: link.from, zone: AXIAL_DIRECTION_VECTORS[link.direction] };
    const to = { space: link.to, zone: AXIAL_DIRECTION_VECTORS[link.to_direction] };
    neighbors.set(zoneKey(from), to);
    neighbors.set(zoneKey(to), from);
  }
  return neighbors;
}

export function acrossBorder(
  location: RisqTickLocation,
  seams: Map<string, RisqTickLocation>
): RisqTickLocation | undefined {
  if (seams.has(zoneKey(location))) {
    return seams.get(zoneKey(location));
  }
  if (location.zone.x === 0 && location.zone.y === 0) {
    return undefined;
  }
  const neighbor = {
    space: { x: location.space.x + location.zone.x, y: location.space.y + location.zone.y },
    zone: { x: -location.zone.x, y: -location.zone.y },
  };
  return seams.has(zoneKey(neighbor)) ? undefined : neighbor;
}
