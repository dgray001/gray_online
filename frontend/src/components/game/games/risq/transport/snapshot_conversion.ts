import type {
  RisqUnit,
  RisqBuilding,
  RisqResource,
  GameRisq,
  RisqSpace,
  GameRisqScoreEntry,
  RisqPlayerResource,
  RisqPlayer,
  RisqZone,
  SPACE_ZONES_TYPE,
  RisqOrder,
} from '../model/types';
import type {
  GameRisqFromServer,
  RisqPlayerResourcesFromServer,
  RisqPlayerFromServer,
  RisqSpaceFromServer,
  RisqZoneFromServer,
  RisqBuildingFromServer,
  RisqResourceFromServer,
  RisqUnitFromServer,
  RisqOrderFromServer,
} from './snapshot_types';
import { RisqResourceType, RisqUnitType, RisqUnitStance } from '../model/types';
import { err } from '../../../../../scripts/log';
import { ColorRGB } from '../../../../../scripts/color_rgb';
import { serverToRisqTurnReport } from './turn_report';
import { resourceType } from '../rendering/assets/resources';
import { organizeZoneUnits } from '../model/unit_groups';
import { RisqTerrainType } from '../rendering/terrain';

/** One instance per internal_id per snapshot, so player, space and zone containers share the same objects */
declare interface RisqEntityRegistry {
  units: Map<number, RisqUnit>;
  buildings: Map<number, RisqBuilding>;
  resources: Map<number, RisqResource>;
}

function createEntityRegistry(): RisqEntityRegistry {
  return { units: new Map(), buildings: new Map(), resources: new Map() };
}

export function serverToGameRisq(server_game: GameRisqFromServer): GameRisq | undefined {
  if (!server_game) {
    return undefined;
  }
  const registry = createEntityRegistry();
  const spaces: (RisqSpace | undefined)[][] = [];
  for (const server_row of server_game.spaces) {
    const row: (RisqSpace | undefined)[] = [];
    for (const space of server_row) {
      row.push(space ? serverToRisqSpace(space, registry) : undefined);
    }
    spaces.push(row);
  }
  const players = server_game.players.filter((p) => !!p).map((p) => serverToRisqPlayer(p, registry));
  const scores: GameRisqScoreEntry[] = [];
  for (const player of players) {
    scores.push({
      player_id: player.player.player_id,
      nickname: player.player.nickname,
      score: player.score,
      color: player.color,
      eliminated: player.eliminated,
    });
  }
  return {
    game_base: server_game.game_base,
    outcome: server_game.outcome,
    players,
    scores: scores.sort((a, b) => b.score - a.score),
    board_size: server_game.board_size,
    population_limit: server_game.population_limit,
    turn_number: server_game.turn_number,
    spaces,
    giving_orders: server_game.giving_orders,
    regions: server_game.regions,
    background_image: server_game.background_image,
    background_top_left: server_game.background_top_left,
    background_top_right: server_game.background_top_right,
  };
}

/** Converts a server response to frontend resources */
export function serverToRisqResources(
  server_resources: RisqPlayerResourcesFromServer
): Map<RisqResourceType, RisqPlayerResource> {
  return new Map<RisqResourceType, RisqPlayerResource>([
    [
      RisqResourceType.FOOD,
      {
        amount: server_resources.food,
        spending: 0,
        gaining: 0,
        workers: 0,
      },
    ],
    [
      RisqResourceType.WOOD,
      {
        amount: server_resources.wood,
        spending: 0,
        gaining: 0,
        workers: 0,
      },
    ],
    [
      RisqResourceType.STONE,
      {
        amount: server_resources.stone,
        spending: 0,
        gaining: 0,
        workers: 0,
      },
    ],
    [
      RisqResourceType.GOLD,
      {
        amount: server_resources.gold,
        spending: 0,
        gaining: 0,
        workers: 0,
      },
    ],
  ]);
}

/** Converts a server response ot a frontend risq player */
export function serverToRisqPlayer(server_player: RisqPlayerFromServer, registry: RisqEntityRegistry): RisqPlayer {
  let color_split: number[] = server_player.color.split(',').map((c) => parseInt(c.trim()));
  if (color_split.length !== 3) {
    err('Error parsing player color', server_player.color);
    color_split = [0, 0, 0];
  }
  const player: RisqPlayer = {
    player: server_player.player,
    resources: server_player.resources ? serverToRisqResources(server_player.resources) : new Map(),
    buildings: new Map(
      server_player.buildings
        .map((b) => serverToRisqBuilding(b, registry))
        .filter((b) => !!b)
        .map((b) => [b.internal_id, b])
    ),
    units: new Map(
      server_player.units
        .map((u) => serverToRisqUnit(u, registry))
        .filter((u) => !!u)
        .map((u) => [u.internal_id, u])
    ),
    population_limit: server_player.population_limit,
    score: server_player.score,
    color: new ColorRGB(color_split[0], color_split[1], color_split[2]),
    active_orders: server_player.active_orders.map((o) => serverToRisqOrder(o)).filter((o) => !!o),
    orders_submitted: server_player.orders_submitted,
    eliminated: server_player.eliminated,
    researched_techs: new Map(
      Object.entries(server_player.researched_techs).map(([tech_id, researched]) => [Number(tech_id), researched])
    ),
    turn_report: server_player.turn_report ? serverToRisqTurnReport(server_player.turn_report) : undefined,
    planned_foundations: new Map((server_player.planned_foundations ?? []).map((f) => [f.coordinate_key, f])),
    available_mercenaries: server_player.available_mercenaries ?? [],
    auto_renewals: new Map(
      (server_player.auto_renewals ?? []).map((queue): [number, number] => [queue.building_id, queue.count])
    ),
    default_unit_stance: server_player.default_unit_stance ?? RisqUnitStance.DEFENSIVE,
    default_unit_attack_back: server_player.default_unit_attack_back ?? true,
    default_unit_interrupt_current: server_player.default_unit_interrupt_current ?? false,
    default_unit_target_priority: server_player.default_unit_target_priority ?? [],
  };
  return player;
}

/** Converts a server response to a frontend risq space */
export function serverToRisqSpace(server_space: RisqSpaceFromServer, registry: RisqEntityRegistry): RisqSpace {
  const is_explored = 'terrain_id' in server_space;
  const space: RisqSpace = {
    terrain_id: is_explored ? server_space.terrain_id : 0,
    terrain_type: is_explored ? server_space.terrain_type : RisqTerrainType.NONE,
    display_name: is_explored ? server_space.display_name : '',
    coordinate: server_space.coordinate,
    coordinate_key: server_space.coordinate_key,
    visibility: server_space.visibility,
    num_military_units: 0,
    num_villager_units: 0,
    unit_count: 'unit_count' in server_space ? server_space.unit_count : undefined,
    ownership: is_explored ? server_space.ownership : undefined,
    gold_income: is_explored ? server_space.gold_income : undefined,
    // purely frontend fields
    center: { x: 0, y: 0 },
    hovered: false,
    hovered_neighbor: false,
    hovered_row: false,
    clicked: false,
  };
  if (is_explored && server_space.zones) {
    const zones: RisqZone[][] = [];
    for (const server_row of server_space.zones) {
      const row: RisqZone[] = [];
      for (const zone of server_row) {
        row.push(serverToRisqZone(zone, registry));
      }
      zones.push(row);
    }
    space.zones = zones as SPACE_ZONES_TYPE;
  }
  if (is_explored && server_space.resources) {
    space.resources = new Map(
      server_space.resources
        .map((r) => serverToRisqResource(r, registry))
        .filter((r) => !!r)
        .map((r) => [r.internal_id, r])
    );
    space.total_resources = new Map<RisqResourceType, number>();
    for (const resource of space.resources.values()) {
      const resource_type = resourceType(resource);
      const existing_resources = space.total_resources.get(resource_type) ?? 0;
      space.total_resources.set(resource_type, existing_resources + resource.resources_left);
    }
  }
  if (is_explored && server_space.buildings) {
    space.buildings = new Map(
      server_space.buildings
        .map((b) => serverToRisqBuilding(b, registry))
        .filter((b) => !!b)
        .map((b) => [b.internal_id, b])
    );
  }
  if ('units' in server_space && server_space.units) {
    space.units = new Map(
      server_space.units
        .map((u) => serverToRisqUnit(u, registry))
        .filter((u) => !!u)
        .map((u) => [u.internal_id, u])
    );
    space.num_military_units = [...space.units.values()].filter((u) => u.unit_type !== RisqUnitType.ECONOMIC).length;
    space.num_villager_units = [...space.units.values()].filter((u) => u.unit_type === RisqUnitType.ECONOMIC).length;
  }
  return space;
}

/** Converts a server response to a frontend risq zone */
export function serverToRisqZone(server_zone: RisqZoneFromServer, registry: RisqEntityRegistry): RisqZone {
  const units = new Map(
    (server_zone.units ?? [])
      .map((u) => serverToRisqUnit(u, registry))
      .filter((u) => !!u)
      .map((u) => [u.internal_id, u])
  );
  const units_by_type = organizeZoneUnits(units);
  return {
    coordinate: server_zone.coordinate,
    coordinate_key: server_zone.coordinate_key,
    resource: serverToRisqResource(server_zone.resource, registry),
    building: serverToRisqBuilding(server_zone.building, registry),
    corpses: server_zone.corpses ?? [],
    units,
    unit_count: server_zone.unit_count,
    terrain_override: server_zone.terrain_override,
    terrain_override_display_name: server_zone.terrain_override_display_name,
    destroyed_building: server_zone.destroyed_building,
    destroyed_building_turns: server_zone.destroyed_building_turns,
    // purely frontend fields
    hovered: false,
    clicked: false,
    hovered_data: [],
    units_by_type,
    military_units: [...units.values()]
      .filter((u) => u.unit_type !== RisqUnitType.ECONOMIC)
      .sort((a, b) => a.unit_id - b.unit_id)
      .map((u) => u.internal_id),
    economic_units: [...units.values()]
      .filter((u) => u.unit_type === RisqUnitType.ECONOMIC)
      .sort((a, b) => a.unit_id - b.unit_id)
      .map((u) => u.internal_id),
    ownership: server_zone.ownership,
  };
}

export function serverToRisqBuilding(
  server_building: RisqBuildingFromServer | undefined,
  registry: RisqEntityRegistry
): RisqBuilding | undefined {
  if (!server_building) {
    return undefined;
  }
  const existing = registry.buildings.get(server_building.internal_id);
  if (existing) {
    return existing;
  }
  const building: RisqBuilding = {
    ...server_building,
    active_orders: server_building.active_orders.map((o) => serverToRisqOrder(o)).filter((o) => !!o),
    hover_data: {
      ps: { x: 0, y: 0 },
      pe: { x: 0, y: 0 },
    },
  };
  registry.buildings.set(building.internal_id, building);
  return building;
}

/** Converts a server response to a frontend risq resource */
export function serverToRisqResource(
  server_resource: RisqResourceFromServer | undefined,
  registry: RisqEntityRegistry
): RisqResource | undefined {
  if (!server_resource) {
    return undefined;
  }
  const existing = registry.resources.get(server_resource.internal_id);
  if (existing) {
    return existing;
  }
  const resource: RisqResource = {
    ...server_resource,
    hover_data: {
      ps: { x: 0, y: 0 },
      pe: { x: 0, y: 0 },
    },
  };
  registry.resources.set(resource.internal_id, resource);
  return resource;
}

export function serverToRisqUnit(
  server_unit: RisqUnitFromServer | undefined,
  registry: RisqEntityRegistry
): RisqUnit | undefined {
  if (!server_unit) {
    return undefined;
  }
  const existing = registry.units.get(server_unit.internal_id);
  if (existing) {
    return existing;
  }
  const unit: RisqUnit = {
    ...server_unit,
    active_orders: server_unit.active_orders.map((o) => serverToRisqOrder(o)).filter((o) => !!o),
    hover_data: {
      ps: { x: 0, y: 0 },
      pe: { x: 0, y: 0 },
    },
  };
  registry.units.set(unit.internal_id, unit);
  return unit;
}

/** Converts a server response to a frontend risq order */
export function serverToRisqOrder(server_order?: RisqOrderFromServer): RisqOrder | undefined {
  if (!server_order) {
    return undefined;
  }
  const order: RisqOrder = {
    ...server_order,
    internal_id: server_order.internal_id ?? 0,
  };
  return order;
}
