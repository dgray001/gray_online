import {ColorRGB} from '../../../../scripts/color_rgb';
import {capitalize} from '../../../../scripts/util';
import {GameBase, GamePlayer} from '../../data_models';
import {Point2D} from '../../util/objects2d';
import {resourceType} from './risq_resources';
import {organizeZoneUnits} from './risq_zone';

/** Data describing a game of risq */
export declare interface GameRisq {
  game_base: GameBase;
  players: RisqPlayer[];
  scores: GameRisqScoreEntry[];
  board_size: number;
  population_limit: number;
  turn_number: number;
  spaces: RisqSpace[][];
}

/** Data describing an entry in the scores array */
export declare interface GameRisqScoreEntry {
  player_id: number;
  nickname: string;
  score: number;
  color: ColorRGB;
}

/** Data describing a risq player */
export declare interface RisqPlayer {
  player: GamePlayer;
  buildings: Map<number, RisqBuilding>; // key is internal_id
  units: Map<number, RisqUnit>; // key is internal_id
  resources: Map<RisqResourceType, number>;
  population_limit: number;
  score: number;
  color: ColorRGB;
}

/** All the resource types */
export enum RisqResourceType {
  ERROR,
  FOOD,
  WOOD,
  STONE,
  GOLD,
}

/** All the terrain types */
export enum RisqTerrainType {
  FLATLANDS = 0,
  HILLY = 1,
  MOUNTAINOUS = 2,
  VALLEY = 3,
  SWAMP = 4,
  SHALLOWS = 5,
  WATER = 6,
  DEEP_WATER = 7,
}

/** Converts terrain type to a string */
export function risqTerrainName(terrain: RisqTerrainType): string {
  return capitalize(RisqTerrainType[terrain].replace('_', ' ').toLowerCase());
}

export const ZONE_VISIBILITY = 3; // can see zones

/** Data describing a hexagonal space in risq */
export declare interface RisqSpace {
  terrain: RisqTerrainType;
  coordinate: Point2D;
  visibility: number; // See risq_vision.go for value meanings
  zones?: RisqZone[][];
  resources?: Map<number, RisqResource>;
  buildings?: Map<number, RisqBuilding>;
  units?: Map<number, RisqUnit>;
  // purely frontend fields
  center: Point2D;
  hovered: boolean;
  hovered_neighbor: boolean;
  hovered_row: boolean;
  clicked: boolean;
  num_military_units?: number;
  num_villager_units?: number;
  total_resources?: Map<RisqResourceType, number>;
}

/** Describes rectangle hover data */
export declare interface RectHoverData {
  ps: Point2D;
  pe: Point2D;
  hovered?: boolean;
  clicked?: boolean;
}

/** Describes ellipse hover data */
export declare interface EllipHoverData {
  c: Point2D;
  r: Point2D;
  hovered?: boolean;
  clicked?: boolean;
}

/** Data describing zones inside a risq space */
export declare interface RisqZone {
  coordinate: Point2D;
  resource?: RisqResource;
  building?: RisqBuilding;
  units: Map<number, RisqUnit>; // <internal_ids, unit>
  // purely frontend fields
  hovered: boolean;
  clicked: boolean;
  hovered_data: EllipHoverData[];
  units_by_type: Map<number, Map<number, UnitByTypeData>>; // <player_id, <unit_id, internal_ids>>
  economic_units_by_type: Map<number, UnitByTypeData[]>; // <player_id, units_by_type>
  military_units_by_type: Map<number, UnitByTypeData[]>; // <player_id, units_by_type>
  economic_units: number[]; // internal_id[]
  military_units: number[]; // internal_id[]
  reset_hovered_data?: boolean;
}

/** Data describing units_by_type data */
export declare interface UnitByTypeData {
  player_id: number;
  unit_id: number;
  units: Set<number>; // internal ids
  hover_data?: RectHoverData;
}

/** Data describing a risq unit */
export declare interface RisqUnit {
  internal_id: number;
  player_id: number;
  unit_id: number;
  display_name: string;
  space_coordinate: Point2D;
  zone_coordinate: Point2D;
  speed: number;
  combat_stats: RisqCombatStats;
  // purely frontend fields
  hover_data: RectHoverData;
}

/** Data describing a risq building */
export declare interface RisqBuilding {
  internal_id: number;
  player_id: number;
  building_id: number;
  display_name: string;
  space_coordinate: Point2D;
  zone_coordinate: Point2D;
  population_support: number;
  combat_stats: RisqCombatStats;
  // purely frontend fields
  hover_data: RectHoverData;
}

/** Data describing combat stats */
export declare interface RisqCombatStats {
  health: number;
  max_health: number;
  attack_type: RisqAttackType;
  attack_blunt: number;
  attack_piercing: number;
  attack_magic: number;
  defense_blunt: number;
  defense_piercing: number;
  defense_magic: number;
  penetration_blunt: number;
  penetration_piercing: number;
  penetration_magic: number;
}

/** All the attack types */
export enum RisqAttackType {
  NONE = 0,
  BLUNT = 1,
  PIERCING = 2,
  MAGIC = 3,
  BLUNT_PIERCING = 4,
  PIERCING_MAGIC = 5,
  MAGIC_BLUNT = 6,
  BLUNT_PIERCING_MAGIC = 7,
}

/** Data describing resources in a zone */
export declare interface RisqResource {
  internal_id: number;
  resource_id: number;
  display_name: string;
  space_coordinate: Point2D;
  zone_coordinate: Point2D;
  resources_left: number;
  base_gather_speed: number;
  // purely frontend fields
  hover_data: RectHoverData; // left panel
}

/** Data describing a game of risq as returned by server */
export declare interface GameRisqFromServer {
  game_base: GameBase; // already been converted
  players: RisqPlayerFromServer[];
  board_size: number;
  population_limit: number;
  turn_number: number;
  spaces: RisqSpaceFromServer[][];
}

/** Data describing risq player resources from server */
export declare interface RisqPlayerResourcesFromServer {
  wood: number;
  food: number;
  stone: number;
}

/** Data describing a risq player */
export declare interface RisqPlayerFromServer {
  player: GamePlayer;
  buildings: RisqBuildingFromServer[];
  units: RisqUnitFromServer[];
  resources: RisqPlayerResourcesFromServer;
  population_limit: number;
  score: number;
  color: string;
}

/** Data describing a hexagonal space in risq */
export declare interface RisqSpaceFromServer {
  terrain: RisqTerrainType;
  coordinate: Point2D;
  visibility: number;
  zones?: RisqZoneFromServer[][];
  resources?: RisqResourceFromServer[];
  buildings?: RisqBuildingFromServer[];
  units?: RisqUnitFromServer[];
}

/** Data describing zones inside a risq space */
export declare interface RisqZoneFromServer {
  coordinate: Point2D;
  building?: RisqBuildingFromServer;
  resource?: RisqResourceFromServer;
  units: RisqUnitFromServer[];
}

/** Data describing a risq unit */
export declare interface RisqUnitFromServer {
  internal_id: number;
  player_id: number;
  unit_id: number;
  display_name: string;
  space_coordinate: Point2D;
  zone_coordinate: Point2D;
  speed: number;
  combat_stats: RisqCombatStatsFromServer;
}

/** Data describing a risq building */
export declare interface RisqBuildingFromServer {
  internal_id: number;
  player_id: number;
  building_id: number;
  display_name: string;
  space_coordinate: Point2D;
  zone_coordinate: Point2D;
  population_support: number;
  combat_stats: RisqCombatStatsFromServer;
}

/** Data describing combat stats */
export declare interface RisqCombatStatsFromServer {
  health: number;
  max_health: number;
  attack_type: RisqAttackType;
  attack_blunt: number;
  attack_piercing: number;
  attack_magic: number;
  defense_blunt: number;
  defense_piercing: number;
  defense_magic: number;
  penetration_blunt: number;
  penetration_piercing: number;
  penetration_magic: number;
}

/** Data describing resources in a zone */
export declare interface RisqResourceFromServer {
  internal_id: number;
  resource_id: number;
  display_name: string;
  space_coordinate: Point2D;
  zone_coordinate: Point2D;
  resources_left: number;
  base_gather_speed: number;
}

/** Converts a server response to a frontend risq game */
export function serverToGameRisq(server_game: GameRisqFromServer): GameRisq {
  if (!server_game) {
    return undefined;
  }
  const spaces: RisqSpace[][] = [];
  for (const server_row of server_game.spaces) {
    const row: RisqSpace[] = [];
    for (const space of server_row) {
      row.push(serverToRisqSpace(space));
    }
    spaces.push(row);
  }
  const players = server_game.players.map(p => serverToRisqPlayer(p));
  const scores: GameRisqScoreEntry[] = [];
  for (const player of players) {
    scores.push({
      player_id: player.player.player_id,
      nickname: player.player.nickname,
      score: player.score,
      color: player.color,
    });
  }
  return {
    game_base: server_game.game_base,
    players,
    scores: scores.sort((a, b) => a.score - b.score),
    board_size: server_game.board_size,
    population_limit: server_game.population_limit,
    turn_number: server_game.turn_number,
    spaces,
  };
}

/** Converts a server response to frontend resources */
export function serverToRisqResources(server_resources: RisqPlayerResourcesFromServer): Map<RisqResourceType, number> {
  return new Map<RisqResourceType, number>([
    [RisqResourceType.FOOD, server_resources.food],
    [RisqResourceType.WOOD, server_resources.wood],
    [RisqResourceType.STONE, server_resources.stone],
  ]);
}

/** Converts a server response ot a frontend risq player */
export function serverToRisqPlayer(server_player: RisqPlayerFromServer): RisqPlayer {
  if (!server_player) {
    return undefined;
  }
  let color_split: number[] = server_player.color.split(',').map(c => parseInt(c.trim()));
  if (color_split.length !== 3) {
    console.error('Error parsing player color', server_player.color);
    color_split = [0, 0, 0];
  }
  const player: RisqPlayer = {
    player: server_player.player,
    resources: serverToRisqResources(server_player.resources),
    buildings: new Map(server_player.buildings.map(b => [b.internal_id, serverToRisqBuilding(b)])),
    units: new Map(server_player.units.map(u => [u.internal_id, serverToRisqUnit(u)])),
    population_limit: server_player.population_limit,
    score: server_player.score,
    color: new ColorRGB(color_split[0], color_split[1], color_split[2]),
  };
  return player;
}

/** Converts a server response to a frontend risq space */
export function serverToRisqSpace(server_space: RisqSpaceFromServer): RisqSpace {
  if (!server_space) {
    return undefined;
  }
  const space: RisqSpace = {
    terrain: server_space.terrain,
    coordinate: server_space.coordinate,
    visibility: server_space.visibility,
    num_military_units: 0,
    num_villager_units: 0,
    // purely frontend fields
    center: {x: 0, y: 0},
    hovered: false,
    hovered_neighbor: false,
    hovered_row: false,
    clicked: false,
  };
  if (!!server_space.zones) {
    const zones: RisqZone[][] = [];
    for (const server_row of server_space.zones) {
      const row: RisqZone[] = [];
      for (const zone of server_row) {
        row.push(serverToRisqZone(zone));
      }
      zones.push(row);
    }
    space.zones = zones;
  }
  if (!!server_space.resources) {
    space.resources = new Map(server_space.resources.map(server_resource =>
      [server_resource.internal_id, serverToRisqResource(server_resource)]));
    space.total_resources = new Map<RisqResourceType, number>();
    for (const resource of space.resources.values()) {
      const resource_type = resourceType(resource);
      if (space.total_resources.has(resource_type)) {
        space.total_resources.set(resource_type, space.total_resources.get(resource_type) + resource.resources_left);
      } else {
        space.total_resources.set(resource_type, resource.resources_left);
      }
    }
  }
  if (!!server_space.buildings) {
    space.buildings = new Map(server_space.buildings.map(server_building =>
      [server_building.internal_id, serverToRisqBuilding(server_building)]));
  }
  if (!!server_space.units) {
    space.units = new Map(server_space.units.map(server_unit =>
      [server_unit.internal_id, serverToRisqUnit(server_unit)]));
    space.num_military_units = [...space.units.values()].filter(u => u.unit_id > 10).length;
    space.num_villager_units = [...space.units.values()].filter(u => u.unit_id < 11).length;
  }
  return space;
}

/** Converts a server response to a frontend risq zone */
export function serverToRisqZone(server_zone: RisqZoneFromServer): RisqZone {
  if (!server_zone) {
    return undefined;
  }
  const units = new Map(server_zone.units.map(server_unit => [server_unit.internal_id, serverToRisqUnit(server_unit)]));
  const units_by_type = organizeZoneUnits(units);
  const economic_units_by_type = new Map<number, UnitByTypeData[]>();
  const military_units_by_type = new Map<number, UnitByTypeData[]>();
  for (const [player_id, player_units] of units_by_type.entries()) {
    economic_units_by_type.set(player_id, []);
    military_units_by_type.set(player_id, []);
    for (const [unit_id, units] of player_units.entries()) {
      if (unit_id < 11) {
        economic_units_by_type.get(player_id).push(units);
      } else {
        military_units_by_type.get(player_id).push(units);
      }
    }
    if (economic_units_by_type.get(player_id).length === 0) {
      economic_units_by_type.delete(player_id);
    }
    if (military_units_by_type.get(player_id).length === 0) {
      military_units_by_type.delete(player_id);
    }
  }
  return {
    coordinate: server_zone.coordinate,
    resource: serverToRisqResource(server_zone.resource),
    building: serverToRisqBuilding(server_zone.building),
    units,
    // purely frontend fields
    hovered: false,
    clicked: false,
    hovered_data: [],
    units_by_type,
    economic_units_by_type,
    military_units_by_type,
    military_units: [...units.values()].filter(u => u.unit_id > 10).sort((a, b) => a.unit_id - b.unit_id).map(u => u.internal_id),
    economic_units: [...units.values()].filter(u => u.unit_id < 11).sort((a, b) => a.unit_id - b.unit_id).map(u => u.internal_id),
  };
}

/** Converts a server response to a frontend risq zone */
export function serverToRisqBuilding(server_building: RisqBuildingFromServer): RisqBuilding {
  if (!server_building) {
    return undefined;
  }
  (server_building as RisqBuilding).hover_data = {ps: {x: 0, y: 0}, pe: {x: 0, y: 0}};
  return server_building as RisqBuilding;
}

/** Converts a server response to a frontend risq resource */
export function serverToRisqResource(server_resource: RisqResourceFromServer): RisqResource {
  if (!server_resource) {
    return undefined;
  }
  (server_resource as RisqResource).hover_data = {ps: {x: 0, y: 0}, pe: {x: 0, y: 0}};
  return server_resource as RisqResource;
}

/** Converts a server response to a frontend risq zone */
export function serverToRisqUnit(server_unit: RisqUnitFromServer): RisqUnit {
  if (!server_unit) {
    return undefined;
  }
  (server_unit as RisqUnit).hover_data = {ps: {x: 0, y: 0}, pe: {x: 0, y: 0}};
  return server_unit as RisqUnit;
}

/** Returns the space from the input index, if the space exists */
export function getSpace(game: GameRisq, index: Point2D): RisqSpace|undefined {
  if (index.x < 0 || index.x >= game.spaces.length) {
    return undefined;
  }
  const row = game.spaces[index.x];
  if (!row) {
    console.log(game.spaces, index.x);
    return undefined;
  }
  if (index.y < 0 || index.y >= row.length) {
    return undefined;
  }
  return row[index.y];
}

/** Transforms the input coordinate in axial space to index space */
export function coordinateToIndex(board_size: number, coordinate: Point2D): Point2D {
  return {
    x: coordinate.y + board_size,
    y: coordinate.x - Math.max(-board_size, -(board_size + coordinate.y)),
  };
}

/** Transforms the input coordinate in index space to axial space */
export function indexToCoordinate(board_size: number, index: Point2D): Point2D {
  const cy = index.x - board_size;
  return {
    x: index.y + Math.max(-board_size, -(board_size + cy)),
    y: cy,
  };
}

/** Data describing a start-turn update */
export declare interface StartTurnData {
  game: GameRisqFromServer;
}
