import type { Point2D } from '../../../../util/objects2d';
import { rotatePoint } from '../../../../util/objects2d';
import type { RisqTickLocation } from '../../model/tick_actions';
import { cantorPair, invertPair } from '../../model/coordinates';
import type { ReplayEvent } from '../../model/replay_event';
import type { GameRisq, RisqRegion, RisqSpace } from '../../model/types';

type EventMap = { spaces: Map<number, RisqSpace>; regions: Map<number, RisqRegion>; rotation: number };
type EventPlace = { name?: string; battle: string };

export function nameReplayEvents(events: ReplayEvent[], game: GameRisq): ReplayEvent[] {
  const map: EventMap = { spaces: new Map(), regions: new Map(), rotation: game.rotation };
  for (const space of game.spaces.flat()) {
    if (space) {
      map.spaces.set(space.coordinate_key, space);
    }
  }
  for (const region of game.regions) {
    if (!region.name) {
      continue;
    }
    for (const key of region.spaces) {
      map.regions.set(key, region);
    }
  }
  return events.map((event: ReplayEvent): ReplayEvent => {
    if (event.kind === 'continent') {
      return event;
    }
    const keys = [
      ...new Set(
        event.locations.map((location: RisqTickLocation): number => cantorPair(location.space.x, location.space.y))
      ),
    ];
    const place = eventPlace(keys, map);
    return { ...event, location_name: place.name, title: event.kind === 'battle' ? place.battle : event.title };
  });
}

function cartesian(key: number, rotation: number): Point2D {
  const point = invertPair(key);
  return rotatePoint({ x: point.x + point.y / 2, y: (Math.sqrt(3) * point.y) / 2 }, rotation);
}

function center(points: Point2D[]): Point2D {
  return {
    x: points.reduce((sum: number, point: Point2D): number => sum + point.x, 0) / points.length,
    y: points.reduce((sum: number, point: Point2D): number => sum + point.y, 0) / points.length,
  };
}

function eventPlace(keys: number[], map: EventMap): EventPlace {
  const custom = keys.length === 1 ? map.spaces.get(keys[0])?.custom_name?.trim() : undefined;
  if (custom) {
    return { name: custom, battle: `Battle of ${custom}` };
  }
  const counts = new Map<RisqRegion, number>();
  for (const key of keys) {
    const region = map.regions.get(key);
    if (region) {
      counts.set(region, (counts.get(region) ?? 0) + 1);
    }
  }
  const ranked = [...counts].sort(
    (a: [RisqRegion, number], b: [RisqRegion, number]): number => b[1] - a[1] || a[0].name.localeCompare(b[0].name)
  );
  if (ranked.length === 0) {
    return { battle: 'Battle' };
  }
  const [region, count] = ranked[0];
  if (count / keys.length >= 2 / 3) {
    const name = regionPart(
      region,
      keys.filter((key: number): boolean => map.regions.get(key) === region),
      map.rotation
    );
    return { name, battle: `Battle of ${name}` };
  }
  return crossRegionPlace(ranked.map(([area]: [RisqRegion, number]): string => area.name));
}

function crossRegionPlace(names: string[]): EventPlace {
  if (names.length === 1) {
    return { name: names[0], battle: `Battle near ${names[0]}` };
  }
  const name =
    names.length === 2 ? names.join(' and ') : `${names.slice(0, -1).join(', ')} and ${names[names.length - 1]}`;
  return { name, battle: `Battle across ${name}` };
}

function axisDirection(values: number[], position: number, low: string, high: string): string {
  const min = Math.min(...values);
  const max = Math.max(...values);
  if (max - min < 1e-6) {
    return '';
  }
  if (position < min + (max - min) / 3) {
    return low;
  }
  return position > max - (max - min) / 3 ? high : '';
}

export function regionPart(region: RisqRegion, keys: number[], rotation: number): string {
  if (region.has_unexplored_spaces || region.spaces.length <= 3 || keys.length / region.spaces.length >= 0.5) {
    return region.name;
  }
  const points = region.spaces.map((key: number): Point2D => cartesian(key, rotation));
  const position = center(keys.map((key: number): Point2D => cartesian(key, rotation)));
  const horizontal = axisDirection(
    points.map((point: Point2D): number => point.x),
    position.x,
    'West',
    'East'
  );
  const vertical = axisDirection(
    points.map((point: Point2D): number => point.y),
    position.y,
    'North',
    'South'
  );
  const direction =
    horizontal && vertical
      ? `${vertical}${horizontal.toLowerCase()}ern`
      : horizontal || vertical
        ? `${horizontal || vertical}ern`
        : 'Central';
  return `${direction} ${region.name}`;
}
