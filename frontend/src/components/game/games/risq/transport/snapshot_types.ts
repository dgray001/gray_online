import type { GameBase, GamePlayer } from '../../../data_models';
import type { RisqTurnReportFromServer } from './turn_report';
import type {
  RisqOutcome,
  RisqCost,
  RisqGatherPoint,
  RisqPlannedFoundation,
  RisqProducible,
  RisqMovePathStep,
  RisqProductionQueueItem,
} from '../model/types';
import type { RisqTerrainType } from '../rendering/terrain';
import type { Point2D } from '../../../util/objects2d';
import type {
  RisqUnitType,
  RisqRange,
  RisqUnitStance,
  RisqTargetCategory,
  RisqAttackType,
  RisqResourceType,
} from '../model/types';

/** Data describing a game of risq as returned by server */
export declare interface GameRisqFromServer {
  game_base: GameBase; // already been converted
  outcome?: RisqOutcome;
  players: RisqPlayerFromServer[];
  board_size: number;
  population_limit: number;
  turn_number: number;
  spaces: (RisqSpaceFromServer | undefined)[][];
  space_links: { from: Point2D; to: Point2D; direction: number }[];
  giving_orders: boolean;
  regions: RisqRegionFromServer[];
  background_image?: string;
  background_top_left?: [number, number];
  background_top_right?: [number, number];
}

/** Data describing a region, as returned by the server */
export declare interface RisqRegionFromServer {
  name: string;
  gold_bonus: number;
  spaces: number[];
  owner: number;
}

/** Data describing risq player resources from server */
export declare interface RisqPlayerResourcesFromServer {
  wood: number;
  food: number;
  stone: number;
  gold: number;
}

/** Data describing a risq player */
export declare interface RisqPlayerFromServer {
  player: GamePlayer;
  buildings: RisqBuildingFromServer[];
  units: RisqUnitFromServer[];
  resources?: RisqPlayerResourcesFromServer;
  population_limit: number;
  max_population_limit: number;
  score: number;
  color: string;
  active_orders: RisqOrderFromServer[];
  orders_submitted: boolean;
  eliminated: boolean;
  researched_techs: Record<string, boolean>;
  turn_report?: RisqTurnReportFromServer;
  planned_foundations?: RisqPlannedFoundation[];
  available_mercenaries?: RisqProducible[];
}

export declare interface RisqSpaceBaseFromServer {
  coordinate: Point2D;
  coordinate_key: number;
}

export declare interface RisqSpaceUnexploredFromServer extends RisqSpaceBaseFromServer {
  visibility: 0; // RisqVisibilityLevel.UNEXPLORED
}

export declare interface RisqSpaceExploredBaseFromServer extends RisqSpaceBaseFromServer {
  terrain_id: number;
  terrain_type: RisqTerrainType;
  display_name: string;
  gold_income: number;
  ownership?: number;
  zones: RisqZoneFromServer[][];
  resources: RisqResourceFromServer[];
  buildings: RisqBuildingFromServer[];
}

export declare interface RisqSpaceFogFromServer extends RisqSpaceExploredBaseFromServer {
  visibility: 1; // RisqVisibilityLevel.FOG
}

export declare interface RisqSpacePoorFromServer extends RisqSpaceExploredBaseFromServer {
  visibility: 2; // RisqVisibilityLevel.POOR
  unit_count: number;
}

export declare interface RisqSpaceGoodFromServer extends RisqSpaceExploredBaseFromServer {
  visibility: 3 | 4; // RisqVisibilityLevel.GOOD | SPY
  units: RisqUnitFromServer[];
}

export type RisqSpaceFromServer =
  | RisqSpaceUnexploredFromServer
  | RisqSpaceFogFromServer
  | RisqSpacePoorFromServer
  | RisqSpaceGoodFromServer;

/** Data describing zones inside a risq space */
export declare interface RisqZoneFromServer {
  coordinate: Point2D;
  coordinate_key: number;
  building?: RisqBuildingFromServer;
  resource?: RisqResourceFromServer;
  units?: RisqUnitFromServer[];
  unit_count?: number;
  ownership?: number;
  terrain_override: number;
  terrain_override_display_name?: string;
}

/** Data describing a risq unit */
export declare interface RisqUnitFromServer {
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
  combat_stats: RisqCombatStatsFromServer;
  attack_range: RisqRange;
  active_orders: RisqOrderFromServer[];
  builds: RisqProducible[];
  garrisoned_in?: number;
  stance?: RisqUnitStance;
  interrupt_current?: boolean;
  attack_back?: boolean;
  target_priority?: RisqTargetCategory[];
  move_path?: RisqMovePathStep[];
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
  garrison_capacity: number;
  has_garrisoned_units: boolean;
  garrisoned_units?: number[]; // only present to the owner or a viewer with spy-tier vision
  turn_stamina: number;
  current_stamina: number;
  max_stamina: number;
  under_construction: boolean;
  stamina_remaining: number;
  construction_stamina_total: number;
  combat_stats: RisqCombatStatsFromServer;
  attack_range: RisqRange;
  active_orders: RisqOrderFromServer[];
  produces: RisqProducible[];
  production_queue: RisqProductionQueueItem[];
  resources_left?: number;
  resource_capacity?: number;
  renew_stamina?: number;
  gather_capacity?: number;
  base_gather_speed?: number;
  renew_cost?: RisqCost;
  renewing?: boolean;
  resource_category?: RisqResourceType;
  gather_point?: RisqGatherPoint;
  auto_attack?: boolean;
  interrupt_current?: boolean;
  target_priority?: RisqTargetCategory[];
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
  gather_capacity: number;
}

/** Data describing an order as returned by the server */
export declare interface RisqOrderFromServer {
  internal_id?: number;
  player_id: number;
  order_type: number;
  target_id: number;
  subjects: number[];
  clear_previous_orders: boolean;
}

export declare interface StartTurnData {
  game: GameRisqFromServer;
}

export declare interface SubmittedOrdersData {
  game: GameRisqFromServer;
  player_id: number;
}

export declare interface UnsubmittedOrdersData {
  game: GameRisqFromServer;
  player_id: number;
}

export declare interface UnitBehaviorSetData {
  game: GameRisqFromServer;
  internal_ids: number[];
  stance?: RisqUnitStance;
  interrupt_current?: boolean;
  attack_back?: boolean;
  target_priority?: RisqTargetCategory[];
}

export declare interface GatherPointSetData {
  game: GameRisqFromServer;
  building_id: number;
  gather_point?: RisqGatherPoint;
}

export declare interface BuildingBehaviorSetData {
  game: GameRisqFromServer;
  internal_ids: number[];
  auto_attack?: boolean;
  interrupt_current?: boolean;
  target_priority?: RisqTargetCategory[];
}
