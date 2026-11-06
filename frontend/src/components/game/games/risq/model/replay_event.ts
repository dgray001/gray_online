import type { RisqTickLocation, RisqTickTarget } from './tick_actions';
import type { Point2D } from '../../../util/objects2d';
import type { RisqProducibleKind } from './types';

export interface ReplayProduction {
  kind: RisqProducibleKind;
  item_id: number;
  source: 'trained' | 'built' | 'researched' | 'hired';
  row?: number;
}

export interface ReplayHiredUnit {
  unit_id: number;
  display_name: string;
  count: number;
}

export function hiredUnitLabel(unit: ReplayHiredUnit): string {
  return `${unit.count}× ${unit.display_name}`;
}

export interface BattlePlayerStats {
  player_id: number;
  troops: number;
  buildings: number;
  kills: number;
  deaths: number;
  razes: number;
  losses: number;
  unit_damage: number;
  building_damage: number;
}

export interface ReplayEvent {
  kind: 'production' | 'battle' | 'eliminated' | 'disruption' | 'continent';
  tick: number;
  player_id: number;
  title: string;
  location_name?: string;
  icon: string;
  locations: RisqTickLocation[];
  map_position?: Point2D;
  end_tick?: number;
  battle_stats?: BattlePlayerStats[];
  battle_participants?: RisqTickTarget[];
  reason?: string;
  failure_row?: number;
  end_of_turn?: boolean;
  production?: ReplayProduction;
  hired_units?: ReplayHiredUnit[];
  continent?: { name: string; change: 'gained' | 'lost'; gold_bonus: number };
}
