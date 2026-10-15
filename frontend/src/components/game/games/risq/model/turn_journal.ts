import type { Point2D } from '../../../util/objects2d';
import type { RisqTickAction, RisqTickEffect } from './tick_actions';
import type { RisqCombatStats, RisqCost, RisqUnitType, RisqVisibilityLevel } from './types';

export declare interface RisqTickUnitState {
  internal_id: number;
  player_id: number;
  unit_id: number;
  unit_type: RisqUnitType;
  display_name: string;
  space_coordinate: Point2D;
  zone_coordinate: Point2D;
  combat_stats: RisqCombatStats;
  current_stamina: number;
  turn_stamina: number;
  max_stamina: number;
  deleted: boolean;
  garrisoned_in?: number;
}

export declare interface RisqTickBuildingState {
  internal_id: number;
  player_id: number;
  building_id: number;
  display_name: string;
  space_coordinate: Point2D;
  zone_coordinate: Point2D;
  combat_stats: RisqCombatStats;
  current_stamina: number;
  turn_stamina: number;
  under_construction: boolean;
  stamina_remaining: number;
  construction_stamina_total: number;
  resources_left: number;
  renew_stamina_remaining: number;
  deleted: boolean;
}

export declare interface RisqTickResourceState {
  internal_id: number;
  resource_id: number;
  space_coordinate: Point2D;
  zone_coordinate: Point2D;
  resources_left: number;
  max_resources: number;
  deleted: boolean;
}

export declare interface RisqTickPlayerState {
  player_id: number;
  resources: RisqCost;
  researched_techs: Map<number, boolean>;
  eliminated: boolean;
}

export declare interface RisqTickSpaceState {
  coordinate_key: number;
  terrain_id: number;
  visibility: RisqVisibilityLevel;
  ownership?: number;
}

export declare interface RisqTickFrame {
  tick: number;
  terminal: boolean;
  phase: string;
  units: Map<number, RisqTickUnitState>;
  buildings: Map<number, RisqTickBuildingState>;
  resources: Map<number, RisqTickResourceState>;
  players: Map<number, RisqTickPlayerState>;
  spaces: Map<number, RisqTickSpaceState>;
  effects: RisqTickEffect[];
}

export declare interface RisqJournalUnit extends RisqTickUnitState {
  tick_actions: RisqTickAction[];
}

export declare interface RisqJournalBuilding extends RisqTickBuildingState {
  tick_actions: RisqTickAction[];
}

export declare interface RisqTurnJournal {
  schema_version: number;
  turn_number: number;
  tick_count: number;
  baseline: RisqTickFrame;
  frames: RisqTickFrame[];
  boundary_events: RisqTickFrame[];
  units: Map<number, RisqJournalUnit>;
  buildings: Map<number, RisqJournalBuilding>;
}
