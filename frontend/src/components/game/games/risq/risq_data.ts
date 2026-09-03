import { ColorRGB } from '../../../../scripts/color_rgb';
import { capitalize } from '../../../../scripts/util';
import type { GameBase, GamePlayer } from '../../data_models';
import type { Point2D } from '../../util/objects2d';
import { resourceType } from './risq_resources';
import { organizeZoneUnits } from './risq_zone';

/** Data describing a game of risq */
export declare interface GameRisq {
  game_base: GameBase;
  players: RisqPlayer[];
  scores: GameRisqScoreEntry[];
  board_size: number;
  population_limit: number;
  turn_number: number;
  spaces: RisqSpace[][];
  giving_orders: boolean;
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
  resources: Map<RisqResourceType, RisqPlayerResource>;
  population_limit: number;
  score: number;
  color: ColorRGB;
  active_orders: RisqOrder[];
  orders_submitted: boolean;
}

/** Data describing frontend resource state */
export declare interface RisqPlayerResource {
  amount: number;
  spending: number;
  gaining: number;
  workers: number;
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

export const NO_VISIBILITY = 0;
export const TERRAIN_VISIBILITY = 1;
export const SPACE_VISIBILITY = 2;
export const ZONE_VISIBILITY = 3;
export const FULL_VISIBILITY = 4;

export type SPACE_ZONES_TYPE = [[RisqZone, RisqZone], [RisqZone, RisqZone, RisqZone], [RisqZone, RisqZone]];

/** Data describing a hexagonal space in risq */
export declare interface RisqSpace {
  terrain: RisqTerrainType;
  coordinate: Point2D;
  coordinate_key: number;
  visibility: number; // See risq_vision.go for value meanings
  zones?: SPACE_ZONES_TYPE;
  resources?: Map<number, RisqResource>;
  buildings?: Map<number, RisqBuilding>;
  units?: Map<number, RisqUnit>;
  ownership: number;
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
  coordinate_key: number;
  resource?: RisqResource;
  building?: RisqBuilding;
  units: Map<number, RisqUnit>; // <internal_ids, unit>
  ownership: number;
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
  turn_stamina: number;
  current_stamina: number;
  max_stamina: number;
  combat_stats: RisqCombatStats;
  active_orders: RisqOrder[];
  builds: RisqProducible[];
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
  turn_stamina: number;
  current_stamina: number;
  max_stamina: number;
  under_construction: boolean;
  stamina_remaining: number;
  construction_stamina_total: number;
  combat_stats: RisqCombatStats;
  active_orders: RisqOrder[];
  produces: RisqProducible[];
  production_queue: RisqProductionQueueItem[];
  // purely frontend fields
  hover_data: RectHoverData;
}

/** Data describing a single item in a building's production queue */
export declare interface RisqProductionQueueItem {
  item_id: number;
  stamina_remaining: number;
  order_internal_id: number;
}

/** All the kinds a producible entry can be */
export enum RisqProducibleKind {
  NONE,
  UNIT,
  TECH,
  BUILDING,
}

/** Data describing something a building can produce, with its cost already resolved for this player */
export declare interface RisqProducible {
  row: number;
  col: number;
  kind: RisqProducibleKind;
  id: number;
  cost: RisqCost;
  stamina_cost: number;
  display_name: string;
  description: string;
}

/** Data describing a resource cost */
export declare interface RisqCost {
  food: number;
  wood: number;
  stone: number;
}

/** Returns whether the player can currently afford the input cost, accounting for already-queued spending */
export function canAffordCost(player: RisqPlayer, cost: RisqCost): boolean {
  const net = (type: RisqResourceType) => {
    const pr = player.resources.get(type);
    return pr ? pr.amount - pr.spending : 0;
  };
  return (
    net(RisqResourceType.FOOD) >= cost.food &&
    net(RisqResourceType.WOOD) >= cost.wood &&
    net(RisqResourceType.STONE) >= cost.stone
  );
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

/** All the order types */
export enum RisqOrderType {
  NONE = 0,
  OrderType_UnitMoveSpace,
  OrderType_UnitMoveZone,
  OrderType_UnitGather,
  OrderType_UnitBuild,
  OrderType_UnitRepair,
  OrderType_UnitAttackSpace,
  OrderType_UnitAttackZone,
  OrderType_UnitAttackUnit,
  OrderType_UnitAttackBuilding,
  OrderType_UnitDefend,
  OrderType_UnitGarrison,
  OrderType_UnitDelete,
  OrderType_BuildingCreate,
  OrderType_BuildingResearch,
  OrderType_CancelOrder,
  OrderType_CancelFoundation,
}

/** Data describing an order */
export declare interface RisqOrder {
  internal_id: number;
  player_id: number;
  order_type: RisqOrderType;
  target_id: number;
  subjects: number[];
}

/** Data describing a frontend order which may be previously submitted or may be unsubmitted */
export type RisqFrontendOrder = Omit<RisqOrder, 'internal_id'> & {
  internal_id?: number;
  clear_previous_orders?: boolean;
};

/** Data describing a game of risq as returned by server */
export declare interface GameRisqFromServer {
  game_base: GameBase; // already been converted
  players: RisqPlayerFromServer[];
  board_size: number;
  population_limit: number;
  turn_number: number;
  spaces: RisqSpaceFromServer[][];
  giving_orders: boolean;
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
  active_orders: RisqOrderFromServer[];
  orders_submitted: boolean;
}

/** Data describing a hexagonal space in risq */
export declare interface RisqSpaceFromServer {
  terrain: RisqTerrainType;
  coordinate: Point2D;
  coordinate_key: number;
  visibility: number;
  zones?: RisqZoneFromServer[][];
  resources?: RisqResourceFromServer[];
  buildings?: RisqBuildingFromServer[];
  units?: RisqUnitFromServer[];
  ownership: number;
}

/** Data describing zones inside a risq space */
export declare interface RisqZoneFromServer {
  coordinate: Point2D;
  coordinate_key: number;
  building?: RisqBuildingFromServer;
  resource?: RisqResourceFromServer;
  units: RisqUnitFromServer[];
  ownership: number;
}

/** Data describing a risq unit */
export declare interface RisqUnitFromServer {
  internal_id: number;
  player_id: number;
  unit_id: number;
  display_name: string;
  space_coordinate: Point2D;
  zone_coordinate: Point2D;
  turn_stamina: number;
  current_stamina: number;
  max_stamina: number;
  combat_stats: RisqCombatStatsFromServer;
  active_orders: RisqOrderFromServer[];
  builds: RisqProducible[];
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
  turn_stamina: number;
  current_stamina: number;
  max_stamina: number;
  under_construction: boolean;
  stamina_remaining: number;
  construction_stamina_total: number;
  combat_stats: RisqCombatStatsFromServer;
  active_orders: RisqOrderFromServer[];
  produces: RisqProducible[];
  production_queue: RisqProductionQueueItem[];
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

/** Data describing an order as returned by the server */
export declare interface RisqOrderFromServer {
  internal_id: number;
  player_id: number;
  order_type: number;
  target_id: number;
  subjects: number[];
}

/** Converts a server response to a frontend risq game */
export function serverToGameRisq(server_game: GameRisqFromServer): GameRisq | undefined {
  if (!server_game) {
    return undefined;
  }
  const spaces: RisqSpace[][] = [];
  for (const server_row of server_game.spaces) {
    const row: RisqSpace[] = [];
    for (const space of server_row) {
      if (!space) {
        continue;
      }
      row.push(serverToRisqSpace(space));
    }
    spaces.push(row);
  }
  const players = server_game.players.filter((p) => !!p).map((p) => serverToRisqPlayer(p));
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
    giving_orders: server_game.giving_orders,
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
  ]);
}

/** Converts a server response ot a frontend risq player */
export function serverToRisqPlayer(server_player: RisqPlayerFromServer): RisqPlayer {
  let color_split: number[] = server_player.color.split(',').map((c) => parseInt(c.trim()));
  if (color_split.length !== 3) {
    console.error('Error parsing player color', server_player.color);
    color_split = [0, 0, 0];
  }
  const player: RisqPlayer = {
    player: server_player.player,
    resources: serverToRisqResources(server_player.resources),
    buildings: new Map(
      server_player.buildings
        .map((b) => serverToRisqBuilding(b))
        .filter((b) => !!b)
        .map((b) => [b.internal_id, b])
    ),
    units: new Map(
      server_player.units
        .map((u) => serverToRisqUnit(u))
        .filter((u) => !!u)
        .map((u) => [u.internal_id, u])
    ),
    population_limit: server_player.population_limit,
    score: server_player.score,
    color: new ColorRGB(color_split[0], color_split[1], color_split[2]),
    active_orders: server_player.active_orders.map((o) => serverToRisqOrder(o)).filter((o) => !!o),
    orders_submitted: server_player.orders_submitted,
  };
  return player;
}

/** Converts a server response to a frontend risq space */
export function serverToRisqSpace(server_space: RisqSpaceFromServer): RisqSpace {
  const space: RisqSpace = {
    terrain: server_space.terrain,
    coordinate: server_space.coordinate,
    coordinate_key: server_space.coordinate_key,
    visibility: server_space.visibility,
    num_military_units: 0,
    num_villager_units: 0,
    ownership: server_space.ownership,
    // purely frontend fields
    center: { x: 0, y: 0 },
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
    space.zones = zones as SPACE_ZONES_TYPE;
  }
  if (!!server_space.resources) {
    space.resources = new Map(
      server_space.resources
        .map((r) => serverToRisqResource(r))
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
  if (!!server_space.buildings) {
    space.buildings = new Map(
      server_space.buildings
        .map((b) => serverToRisqBuilding(b))
        .filter((b) => !!b)
        .map((b) => [b.internal_id, b])
    );
  }
  if (!!server_space.units) {
    space.units = new Map(
      server_space.units
        .map((u) => serverToRisqUnit(u))
        .filter((u) => !!u)
        .map((u) => [u.internal_id, u])
    );
    space.num_military_units = [...space.units.values()].filter((u) => u.unit_id > 10).length;
    space.num_villager_units = [...space.units.values()].filter((u) => u.unit_id < 11).length;
  }
  return space;
}

/** Converts a server response to a frontend risq zone */
export function serverToRisqZone(server_zone: RisqZoneFromServer): RisqZone {
  const units = new Map(
    server_zone.units
      .map((u) => serverToRisqUnit(u))
      .filter((u) => !!u)
      .map((u) => [u.internal_id, u])
  );
  const units_by_type = organizeZoneUnits(units);
  const economic_units_by_type = new Map<number, UnitByTypeData[]>();
  const military_units_by_type = new Map<number, UnitByTypeData[]>();
  for (const [player_id, player_units] of units_by_type.entries()) {
    economic_units_by_type.set(player_id, []);
    military_units_by_type.set(player_id, []);
    for (const [unit_id, units] of player_units.entries()) {
      if (unit_id < 11) {
        economic_units_by_type.get(player_id)!.push(units);
      } else {
        military_units_by_type.get(player_id)!.push(units);
      }
    }
    if (economic_units_by_type.get(player_id)!.length === 0) {
      economic_units_by_type.delete(player_id);
    }
    if (military_units_by_type.get(player_id)!.length === 0) {
      military_units_by_type.delete(player_id);
    }
  }
  return {
    coordinate: server_zone.coordinate,
    coordinate_key: server_zone.coordinate_key,
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
    military_units: [...units.values()]
      .filter((u) => u.unit_id > 10)
      .sort((a, b) => a.unit_id - b.unit_id)
      .map((u) => u.internal_id),
    economic_units: [...units.values()]
      .filter((u) => u.unit_id < 11)
      .sort((a, b) => a.unit_id - b.unit_id)
      .map((u) => u.internal_id),
    ownership: server_zone.ownership,
  };
}

/** Converts a server response to a frontend risq zone */
export function serverToRisqBuilding(server_building?: RisqBuildingFromServer): RisqBuilding | undefined {
  if (!server_building) {
    return undefined;
  }
  const building: RisqBuilding = {
    ...server_building,
    active_orders: server_building.active_orders.map((o) => serverToRisqOrder(o)).filter((o) => !!o),
    hover_data: {
      ps: { x: 0, y: 0 },
      pe: { x: 0, y: 0 },
    },
  };
  return building;
}

/** Converts a server response to a frontend risq resource */
export function serverToRisqResource(server_resource?: RisqResourceFromServer): RisqResource | undefined {
  if (!server_resource) {
    return undefined;
  }
  (server_resource as RisqResource).hover_data = {
    ps: { x: 0, y: 0 },
    pe: { x: 0, y: 0 },
  };
  return server_resource as RisqResource;
}

/** Converts a server response to a frontend risq zone */
export function serverToRisqUnit(server_unit?: RisqUnitFromServer): RisqUnit | undefined {
  if (!server_unit) {
    return undefined;
  }
  const unit: RisqUnit = {
    ...server_unit,
    active_orders: server_unit.active_orders.map((o) => serverToRisqOrder(o)).filter((o) => !!o),
    hover_data: {
      ps: { x: 0, y: 0 },
      pe: { x: 0, y: 0 },
    },
  };
  return unit;
}

/** Converts a server response to a frontend risq order */
export function serverToRisqOrder(server_order?: RisqOrderFromServer): RisqOrder | undefined {
  if (!server_order) {
    return undefined;
  }
  const order: RisqOrder = {
    ...server_order,
  };
  return order;
}
