import type { RisqTickLocation } from './tick_actions';
import type { Point2D } from '../../../util/objects2d';

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
  kind: 'production' | 'battle' | 'eliminated' | 'disruption';
  tick: number;
  player_id: number;
  title: string;
  icon: string;
  locations: RisqTickLocation[];
  map_position?: Point2D;
  end_tick?: number;
  battle_stats?: BattlePlayerStats[];
  reason?: string;
  end_of_turn?: boolean;
}
