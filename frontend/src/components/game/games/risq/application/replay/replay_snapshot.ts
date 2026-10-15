import type { Point2D } from '../../../../util/objects2d';
import { RisqRange } from '../../model/types';
import type {
  RisqTickBuildingState,
  RisqTickFrame,
  RisqTickResourceState,
  RisqTickUnitState,
  RisqTurnJournal,
} from '../../model/turn_journal';
import type {
  GameRisqFromServer,
  RisqBuildingFromServer,
  RisqPlayerFromServer,
  RisqResourceFromServer,
  RisqSpaceFromServer,
  RisqUnitFromServer,
  RisqZoneFromServer,
} from '../../transport/snapshot_types';

declare interface ReplayContext {
  journal: RisqTurnJournal;
  frame: RisqTickFrame;
  units: RisqUnitFromServer[];
  buildings: RisqBuildingFromServer[];
  resources: RisqResourceFromServer[];
  zone_units: Map<string, RisqUnitFromServer[]>;
  zone_buildings: Map<string, RisqBuildingFromServer>;
  zone_resources: Map<string, RisqResourceFromServer>;
}

function zoneKey(space: Point2D, zone: Point2D): string {
  return `${space.x},${space.y},${zone.x},${zone.y}`;
}

function unitFromState(state: RisqTickUnitState, base?: RisqUnitFromServer): RisqUnitFromServer {
  const attack_range = base?.attack_range ?? RisqRange.NONE;
  return {
    ...base,
    ...state,
    attack_range,
    active_orders: [],
    builds: base?.builds ?? [],
    move_path: undefined,
    tick_actions: undefined,
  };
}

function buildingFromState(state: RisqTickBuildingState, base?: RisqBuildingFromServer): RisqBuildingFromServer {
  return {
    population_support: 0,
    garrison_capacity: 0,
    has_garrisoned_units: false,
    max_stamina: state.turn_stamina,
    attack_range: RisqRange.NONE,
    produces: [],
    production_queue: [],
    ...base,
    ...state,
    active_orders: [],
  };
}

function resourceFromState(state: RisqTickResourceState, base?: RisqResourceFromServer): RisqResourceFromServer {
  return { display_name: '', base_gather_speed: 0, gather_capacity: 0, ...base, ...state };
}

function allBase<T extends { internal_id: number }>(
  players: RisqPlayerFromServer[],
  pick: (p: RisqPlayerFromServer) => T[]
): Map<number, T> {
  return new Map(players.flatMap(pick).map((entity): [number, T] => [entity.internal_id, entity]));
}

function buildContext(base: GameRisqFromServer, journal: RisqTurnJournal, frame: RisqTickFrame): ReplayContext {
  const base_units = allBase(base.players, (player) => player.units);
  const base_buildings = allBase(base.players, (player) => player.buildings);
  const units = [...frame.units.values()]
    .filter((state) => !state.deleted)
    .map((state) => unitFromState(state, base_units.get(state.internal_id)));
  const buildings = [...frame.buildings.values()]
    .filter((state) => !state.deleted)
    .map((state) => buildingFromState(state, base_buildings.get(state.internal_id)));
  const resources = [...frame.resources.values()]
    .filter((state) => !state.deleted)
    .map((state) => resourceFromState(state));
  const zone_units = new Map<string, RisqUnitFromServer[]>();
  for (const unit of units.filter((u) => u.garrisoned_in === undefined)) {
    const key = zoneKey(unit.space_coordinate, unit.zone_coordinate);
    zone_units.set(key, [...(zone_units.get(key) ?? []), unit]);
  }
  const by_zone = <T extends { space_coordinate: Point2D; zone_coordinate: Point2D }>(items: T[]): Map<string, T> =>
    new Map(items.map((item): [string, T] => [zoneKey(item.space_coordinate, item.zone_coordinate), item]));
  return {
    journal,
    frame,
    units,
    buildings,
    resources,
    zone_units,
    zone_buildings: by_zone(buildings),
    zone_resources: by_zone(resources),
  };
}

function replayZone(space: Point2D, zone: RisqZoneFromServer, ctx: ReplayContext): RisqZoneFromServer {
  const key = zoneKey(space, zone.coordinate);
  const unseen_building = zone.building && !ctx.journal.buildings.has(zone.building.internal_id);
  return {
    ...zone,
    units: ctx.zone_units.get(key) ?? [],
    building: ctx.zone_buildings.get(key) ?? (unseen_building ? zone.building : undefined),
    resource: ctx.zone_resources.get(key) ?? zone.resource,
    corpses: [],
  };
}

function replaySpace(space: RisqSpaceFromServer, ctx: ReplayContext): RisqSpaceFromServer {
  if (!('zones' in space)) {
    return space;
  }
  const zones = space.zones.map((row) => row.map((zone) => replayZone(space.coordinate, zone, ctx)));
  const flat = zones.flat();
  const units = flat.flatMap((zone) => zone.units ?? []);
  return {
    ...space,
    visibility: Math.max(space.visibility, ctx.frame.spaces.get(space.coordinate_key)?.visibility ?? 0),
    ownership: ctx.frame.spaces.get(space.coordinate_key)?.ownership ?? space.ownership,
    zones,
    units,
    buildings: flat.flatMap((zone) => zone.building ?? []),
    resources: flat.flatMap((zone) => zone.resource ?? []),
  } as RisqSpaceFromServer;
}

function replayPlayer(player: RisqPlayerFromServer, ctx: ReplayContext): RisqPlayerFromServer {
  const id = player.player.player_id;
  const state = ctx.frame.players.get(id);
  return {
    ...player,
    units: ctx.units.filter((unit) => unit.player_id === id),
    buildings: ctx.buildings.filter((building) => building.player_id === id),
    resources: state ? { ...state.resources } : player.resources,
    researched_techs: state ? Object.fromEntries(state.researched_techs) : player.researched_techs,
    active_orders: [],
  };
}

function mergeInto<T>(target: Map<number, T>, source: Map<number, T>): void {
  source.forEach((state: T, id: number): void => void target.set(id, state));
}

/** Frames only hold entities that changed, so the state after a tick is the baseline plus every frame up to it */
function stateAt(journal: RisqTurnJournal, tick: number): RisqTickFrame {
  const merged: RisqTickFrame = {
    ...journal.baseline,
    units: new Map(journal.baseline.units),
    buildings: new Map(journal.baseline.buildings),
    resources: new Map(journal.baseline.resources),
    players: new Map(journal.baseline.players),
    spaces: new Map(journal.baseline.spaces),
  };
  for (const frame of journal.frames.filter((f) => !f.terminal && f.tick <= tick)) {
    mergeInto(merged.units, frame.units);
    mergeInto(merged.buildings, frame.buildings);
    mergeInto(merged.resources, frame.resources);
    mergeInto(merged.players, frame.players);
    mergeInto(merged.spaces, frame.spaces);
  }
  return merged;
}

/** Rewrites the end-of-turn snapshot to the state after the given tick (0 is before tick 1) */
export function replaySnapshot(base: GameRisqFromServer, journal: RisqTurnJournal, tick: number): GameRisqFromServer {
  const frame = stateAt(journal, tick);
  const ctx = buildContext(base, journal, frame);
  return {
    ...base,
    spaces: base.spaces.map((row) => row.map((space) => (space ? replaySpace(space, ctx) : undefined))),
    players: base.players.map((player) => replayPlayer(player, ctx)),
  };
}
