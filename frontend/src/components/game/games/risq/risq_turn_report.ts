import type { Point2D } from '../../util/objects2d';
import type { RisqOrderType } from './risq_data';
import { RisqResourceType } from './risq_data';

/** One row of the turn report's scoreboard */
export declare interface RisqReportScoreLine {
  player_id: number;
  was: number;
  now: number;
}

/** The land section of a turn report */
export declare interface RisqReportLand {
  held_start: number;
  held_end: number;
  gained: number;
  lost: number;
  newly_explored: number;
  gold_from_land: number;
}

/** One resource's row in the turn report's ledger */
export declare interface RisqReportResourceLine {
  resource_type: RisqResourceType;
  start: number;
  gathered: number;
  spent: number;
  final: number;
}

/** A unit type trained this turn, for the turn report */
export declare interface RisqReportUnitCreated {
  unit_id: number;
  count: number;
}

/** A building completed this turn, for the turn report */
export declare interface RisqReportBuildingBuilt {
  building_id: number;
  space: Point2D;
  zone: Point2D;
}

/** All the kinds a combat event in a turn report can be, mirroring the backend's RisqCombatEventKind */
export enum RisqCombatEventKind {
  NONE = 0,
  BUILDING_RAZED = 1,
  BUILDING_LOST = 2,
  UNIT_KILLED = 3,
  UNIT_LOST = 4,
}

/** One combat event that occurred this turn, for the turn report */
export declare interface RisqReportCombatEvent {
  tick: number;
  kind: RisqCombatEventKind;
  self_player: number;
  other_player: number;
  target_id: number;
  space: Point2D;
  zone: Point2D;
  damage: number;
}

/** One order that failed to be received this turn, for the turn report */
export declare interface RisqReportOrderFailure {
  order_type: RisqOrderType;
  target_id: number;
  reason: string;
}

/** The end-of-turn report for a single player */
export declare interface RisqTurnReport {
  turn: number;
  eliminated: boolean;
  scores: RisqReportScoreLine[];
  land: RisqReportLand;
  resources: RisqReportResourceLine[];
  population: { start: number; end: number; cap_start: number; cap_end: number };
  production: {
    units_created: RisqReportUnitCreated[];
    buildings_built: RisqReportBuildingBuilt[];
    techs_researched: number[];
  };
  combat: RisqReportCombatEvent[];
  orders: {
    active: number;
    added: number;
    failed: number;
    executed: number;
    cancelled: number;
    failures: RisqReportOrderFailure[];
  };
}

/** One resource's row in the turn report's ledger, as returned by the server */
export declare interface RisqReportResourceLineFromServer {
  category: number;
  start: number;
  gathered: number;
  spent: number;
  final: number;
}

/** A building completed this turn, as returned by the server */
export declare interface RisqReportBuildingBuiltFromServer {
  building_id: number;
  space: Point2D;
  zone: Point2D;
}

/** One combat event that occurred this turn, as returned by the server */
export declare interface RisqReportCombatEventFromServer {
  tick: number;
  kind: RisqCombatEventKind;
  self_player: number;
  other_player: number;
  target_id: number;
  space: Point2D;
  zone: Point2D;
  damage: number;
}

/** One order that failed to be received this turn, as returned by the server */
export declare interface RisqReportOrderFailureFromServer {
  order_type: RisqOrderType;
  target_id: number;
  reason: string;
}

/** The end-of-turn report for a single player, as returned by the server */
export declare interface RisqTurnReportFromServer {
  turn: number;
  eliminated: boolean;
  scores: RisqReportScoreLine[];
  land: RisqReportLand;
  resources: RisqReportResourceLineFromServer[];
  population: { start: number; end: number; cap_start: number; cap_end: number };
  production: {
    units_created: RisqReportUnitCreated[];
    buildings_built: RisqReportBuildingBuiltFromServer[];
    techs_researched: number[];
  };
  combat: RisqReportCombatEventFromServer[];
  orders: {
    active: number;
    added: number;
    failed: number;
    executed: number;
    cancelled: number;
    failures: RisqReportOrderFailureFromServer[];
  };
}

/** Converts a server response to a frontend turn report */
export function serverToRisqTurnReport(server_report: RisqTurnReportFromServer): RisqTurnReport {
  const resource_order = [RisqResourceType.FOOD, RisqResourceType.WOOD, RisqResourceType.STONE, RisqResourceType.GOLD];
  return {
    ...server_report,
    resources: server_report.resources.map((line, i) => ({
      resource_type: resource_order[i] ?? RisqResourceType.ERROR,
      start: line.start,
      gathered: line.gathered,
      spent: line.spent,
      final: line.final,
    })),
  };
}
