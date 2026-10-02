import type { GameBase, GamePlayer } from '../../../data_models';
import type { RisqTurnReportFromServer } from './turn_report';
import type {
  RisqGatherPoint,
  RisqPlannedFoundation,
  RisqProducible,
  RisqMovePathStep,
  RisqProductionQueueItem,
} from '../model/types';
import type { RisqTerrainType } from '../rendering/terrain';
import type { Point2D } from '../../../util/objects2d';
import type { RisqUnitType, RisqRange, RisqUnitStance, RisqTargetCategory, RisqAttackType } from '../model/types';
/** Data describing a game of risq as returned by server */
export declare interface GameRisqFromServer {
  game_base: GameBase; // already been converted
  players: RisqPlayerFromServer[];
  board_size: number;
  population_limit: number;
  turn_number: number;
  spaces: (RisqSpaceFromServer | undefined)[][];
  giving_orders: boolean;
  regions: RisqRegionFromServer[];
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

/** Data describing a hexagonal space in risq */
export declare interface RisqSpaceFromServer {
  terrain_id?: number;
  terrain_type?: RisqTerrainType;
  display_name?: string;
  coordinate: Point2D;
  coordinate_key: number;
  visibility: number;
  zones?: RisqZoneFromServer[][];
  resources?: RisqResourceFromServer[];
  buildings?: RisqBuildingFromServer[];
  units?: RisqUnitFromServer[];
  unit_count?: number;
  ownership?: number;
  gold_income?: number;
}

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
  internal_id: number;
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
  internal_ids: number[];
  stance?: RisqUnitStance;
  interrupt_current?: boolean;
  attack_back?: boolean;
  target_priority?: RisqTargetCategory[];
}

export declare interface GatherPointSetData {
  building_id: number;
  gather_point?: RisqGatherPoint;
}
