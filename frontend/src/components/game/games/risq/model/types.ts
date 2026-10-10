import type { GameBase, GamePlayer } from '../../../data_models';
import type { ColorRGB } from '../../../../../scripts/color_rgb';
import type { RisqTurnReport } from '../transport/turn_report';
import type { RisqTerrainType } from '../rendering/terrain';
import type { Point2D } from '../../../util/objects2d';

/** Data describing a game of risq */
export declare interface GameRisq {
  game_base: GameBase;
  outcome?: RisqOutcome;
  players: RisqPlayer[];
  scores: GameRisqScoreEntry[];
  board_size: number;
  population_limit: number;
  turn_number: number;
  spaces: (RisqSpace | undefined)[][];
  giving_orders: boolean;
  regions: RisqRegion[];
  background_image?: string;
  background_top_left?: [number, number];
  background_top_right?: [number, number];
}

export declare interface RisqOutcome {
  winner_player_ids: number[];
}

/** Data describing a region: only present for a player if they've explored at least one of its spaces */
export declare interface RisqRegion {
  name: string;
  gold_bonus: number;
  spaces: number[];
  owner: number;
}

/** Data describing an entry in the scores array */
export declare interface GameRisqScoreEntry {
  player_id: number;
  nickname: string;
  score: number;
  color: ColorRGB;
  eliminated: boolean;
}

export declare interface RisqPlannedFoundation {
  coordinate_key: number;
  building_id: number;
  display_name: string;
  stamina_cost: number;
  cost?: RisqCost;
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
  eliminated: boolean;
  researched_techs: Map<number, boolean>;
  turn_report?: RisqTurnReport;
  planned_foundations: Map<number, RisqPlannedFoundation>;
  available_mercenaries: RisqProducible[];
  auto_renewals: Map<number, number>;
  default_unit_stance: RisqUnitStance;
  default_unit_attack_back: boolean;
  default_unit_interrupt_current: boolean;
  default_unit_target_priority: RisqTargetCategory[];
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

export enum RisqVisibilityLevel {
  UNEXPLORED = 0,
  FOG = 1,
  POOR = 2,
  GOOD = 3,
  SPY = 4,
}

export type SPACE_ZONES_TYPE = [[RisqZone, RisqZone], [RisqZone, RisqZone, RisqZone], [RisqZone, RisqZone]];

/** Data describing a hexagonal space in risq */
export declare interface RisqSpace {
  terrain_id: number;
  terrain_type: RisqTerrainType;
  display_name: string;
  coordinate: Point2D;
  coordinate_key: number;
  visibility: number; // See risq_vision.go for value meanings
  zones?: SPACE_ZONES_TYPE;
  resources?: Map<number, RisqResource>;
  buildings?: Map<number, RisqBuilding>;
  units?: Map<number, RisqUnit>;
  unit_count?: number;
  ownership?: number;
  gold_income?: number;
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

export declare interface RisqCorpse {
  internal_id: number;
  player_id: number;
  unit_id: number;
  turns: number;
}

/** Data describing zones inside a risq space */
export declare interface RisqZone {
  coordinate: Point2D;
  coordinate_key: number;
  resource?: RisqResource;
  building?: RisqBuilding;
  corpses: RisqCorpse[];
  units: Map<number, RisqUnit>; // <internal_ids, unit>
  unit_count?: number;
  ownership?: number;
  terrain_override: number;
  terrain_override_display_name?: string;
  destroyed_building?: number;
  destroyed_building_turns?: number;
  // purely frontend fields
  hovered: boolean;
  clicked: boolean;
  hovered_data: EllipHoverData[];
  units_by_type: Map<number, Map<number, UnitByTypeData>>; // <player_id, <unit_id, internal_ids>>
  unit_slots?: UnitByTypeData[][];
  economic_units: number[]; // internal_id[]
  military_units: number[]; // internal_id[]
  economic_hover_data?: RectHoverData;
  military_hover_data?: RectHoverData;
  reset_hovered_data?: boolean;
}

export declare interface UnitByTypeData {
  player_id: number;
  unit_id: number;
  unit_type: RisqUnitType;
  units: Set<number>; // internal ids
  hover_data?: RectHoverData;
}

/** One waypoint in a moving unit's currently planned route */
export declare interface RisqMovePathStep {
  space: Point2D;
  zone: Point2D;
}

export declare interface RisqTickAction {
  tick: number;
  sequence: number;
  order?: Pick<RisqFrontendOrder, 'internal_id' | 'order_type' | 'target_id'>;
  execute: {
    kind: string;
    outcome: 'executed' | 'partial' | 'blocked' | 'skipped';
    target?: { kind?: 'unit' | 'building' | 'resource'; internal_id?: number };
    target_location?: RisqMovePathStep;
  };
}

/** Data describing a risq unit */
export declare interface RisqUnit {
  tick_actions?: RisqTickAction[];
  internal_id: number;
  player_id: number;
  unit_id: number;
  unit_type: RisqUnitType;
  display_name: string;
  space_coordinate: Point2D;
  zone_coordinate: Point2D;
  turn_stamina: number;
  current_stamina: number;
  max_stamina: number;
  combat_stats: RisqCombatStats;
  attack_range: RisqRange;
  active_orders: RisqOrder[];
  builds: RisqProducible[];
  garrisoned_in?: number;
  stance?: RisqUnitStance;
  interrupt_current?: boolean;
  attack_back?: boolean;
  target_priority?: RisqTargetCategory[];
  move_path?: RisqMovePathStep[];
  // purely frontend fields
  hover_data: RectHoverData;
}

/** All the kinds a gather point's location can be, mirroring the backend's RisqGatherPointLocationKind */
export enum RisqGatherPointLocationKind {
  NONE = 0,
  SPACE = 1,
  ZONE = 2,
}

/** All the kinds a gather point's specific object can be, mirroring the backend's RisqGatherObjectType */
export enum RisqGatherObjectType {
  NONE = 0,
  UNIT = 1,
  BUILDING = 2,
  RESOURCE = 3,
}

/** Data describing a building's gather point */
export declare interface RisqGatherPoint {
  location_kind: RisqGatherPointLocationKind;
  location_id: number;
  object_type: RisqGatherObjectType;
  object_id: number;
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
  garrison_capacity: number;
  has_garrisoned_units: boolean;
  garrisoned_units?: number[]; // only present to the owner or a viewer with spy-tier vision
  turn_stamina: number;
  current_stamina: number;
  max_stamina: number;
  under_construction: boolean;
  stamina_remaining: number;
  construction_stamina_total: number;
  combat_stats: RisqCombatStats;
  attack_range: RisqRange;
  active_orders: RisqOrder[];
  produces: RisqProducible[];
  production_queue: RisqProductionQueueItem[];
  resources_left?: number;
  resource_capacity?: number;
  renew_stamina?: number;
  renew_stamina_remaining?: number;
  gather_capacity?: number;
  base_gather_speed?: number;
  renew_cost?: RisqCost;
  renewing?: boolean;
  resource_category?: RisqResourceType;
  gather_point?: RisqGatherPoint; // only present to the owner or a viewer with spy-tier vision
  // purely frontend fields
  hover_data: RectHoverData;
}

/** Data describing a single item in a building's production queue */
export declare interface RisqProductionQueueItem {
  kind: RisqProducibleKind;
  item_id: number;
  stamina_remaining: number;
  order_internal_id: number;
}

/** All the kinds a producible entry can be */
export enum RisqProducibleKind {
  NONE = 0,
  UNIT = 1,
  TECH = 2,
  BUILDING = 3,
}

/** Data describing something a building can produce, with its cost already resolved for this player */
export declare interface RisqProducible {
  stats?: RisqUnitStatsEntry;
  row: number;
  col: number;
  kind: RisqProducibleKind;
  id: number;
  cost: RisqCost;
  stamina_cost: number;
  display_name: string;
  description: string;
  required_tech_id: number;
}

export interface RisqUnitStatsEntry {
  health: number;
  attack_blunt: number;
  attack_piercing: number;
  attack_range: RisqRange;
  defense_blunt: number;
  defense_piercing: number;
  penetration_blunt: number;
  penetration_piercing: number;
}

/** Data describing a resource cost */
export declare interface RisqCost {
  food: number;
  wood: number;
  stone: number;
  gold: number;
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

/** All the unit types */
export enum RisqUnitType {
  NONE = 0,
  ECONOMIC = 1,
  INFANTRY = 2,
  ARCHER = 3,
  CAVALRY = 4,
}

/** All the unit stances */
export enum RisqUnitStance {
  NONE = 0,
  PASSIVE = 1,
  AGGRESSIVE = 2,
  DEFENSIVE = 3,
  STAND_GROUND = 4,
}

/** All the target categories */
export enum RisqTargetCategory {
  NONE = 0,
  ECONOMIC = 1,
  MILITARY = 2,
  BUILDING = 3,
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

/** All the attack ranges, mirroring the backend's RisqRange */
export enum RisqRange {
  NONE = 0,
  ZONE = 1,
  SPACE = 2,
  ADJACENT = 3,
  SECONDARY = 4,
}

/** Static per-resource-id config served by the backend */
export declare interface RisqResourceConfig {
  resource_id: number;
  display_name: string;
  category: RisqResourceType;
  starting_resources: number;
  base_gather_speed: number;
  gather_capacity: number;
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
  gather_capacity: number;
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
  OrderType_UnitRenew,
  OrderType_UnitAttackSpace,
  OrderType_UnitAttackZone,
  OrderType_UnitAttackUnit,
  OrderType_UnitAttackBuilding,
  OrderType_UnitAutoAttackUnit, // Server synthesized; not submitted by player
  OrderType_UnitAutoAttackBuilding, // Server synthesized; not submitted by player
  OrderType_UnitGarrison,
  OrderType_UnitUngarrison,
  OrderType_UnitDelete,
  OrderType_BuildingCreate,
  OrderType_BuildingResearch,
  OrderType_BuildingDelete,
  OrderType_BuildingAttackUnit,
  OrderType_BuildingAttackBuilding,
  OrderType_BuildingAutoAttackUnit, // Server synthesized; not submitted by player
  OrderType_BuildingAutoAttackBuilding, // Server synthesized; not submitted by player
  OrderType_CancelOrder,
  OrderType_CancelFoundation,
  OrderType_BuyMercenary,
}

/** Data describing an order */
export declare interface RisqOrder {
  internal_id: number;
  player_id: number;
  order_type: RisqOrderType;
  target_id: number;
  subjects: number[];
  clear_previous_orders?: boolean;
}

/** Data describing a frontend order which may be previously submitted or may be unsubmitted */
export type RisqFrontendOrder = Omit<RisqOrder, 'internal_id'> & {
  internal_id?: number;
  clear_previous_orders?: boolean;
};

export function canHaveGatherPoint(building: RisqBuilding): boolean {
  return building.garrison_capacity > 0 || building.produces.some((p) => p.kind === RisqProducibleKind.UNIT);
}

/** Returns whether the player can currently afford the input cost, accounting for already-queued spending */
export function canAffordCost(player: RisqPlayer, cost: RisqCost): boolean {
  const net = (type: RisqResourceType): number => {
    const pr = player.resources.get(type);
    return pr ? pr.amount - pr.spending : 0;
  };
  return (
    net(RisqResourceType.FOOD) >= cost.food &&
    net(RisqResourceType.WOOD) >= cost.wood &&
    net(RisqResourceType.STONE) >= cost.stone &&
    net(RisqResourceType.GOLD) >= cost.gold
  );
}

export function meetsTechRequirement(player: RisqPlayer, required_tech_id: number): boolean {
  return required_tech_id === 0 || !!player.researched_techs.get(required_tech_id);
}
