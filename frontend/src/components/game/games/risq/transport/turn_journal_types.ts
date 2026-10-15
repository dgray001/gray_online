import type {
  RisqTickAction,
  RisqTickEffect,
  RisqTickExecution,
  RisqTickIntent,
  RisqTickOrder,
  RisqTickTarget,
} from '../model/tick_actions';
import type { RisqTickBuildingState, RisqTickPlayerState, RisqTickResourceState } from '../model/turn_journal';
import type { RisqTickSpaceState, RisqTickUnitState } from '../model/turn_journal';

/** The server sends an empty object for a target that is absent */
export type RisqTickTargetFromServer = Partial<RisqTickTarget>;

export declare interface RisqTickOrderFromServer extends Omit<RisqTickOrder, 'source_actor'> {
  source_actor?: RisqTickTargetFromServer;
}

export declare interface RisqTickEffectFromServer extends Omit<RisqTickEffect, 'order' | 'actor' | 'target'> {
  order?: RisqTickOrderFromServer;
  actor?: RisqTickTargetFromServer;
  target?: RisqTickTargetFromServer;
}

export declare interface RisqTickIntentFromServer extends Omit<RisqTickIntent, 'target'> {
  target?: RisqTickTargetFromServer;
}

export declare interface RisqTickExecutionFromServer extends Omit<RisqTickExecution, 'cost' | 'target' | 'effects'> {
  food_spent: number;
  wood_spent: number;
  stone_spent: number;
  gold_spent: number;
  target?: RisqTickTargetFromServer;
  effects: RisqTickEffectFromServer[];
}

export declare interface RisqTickActionFromServer extends Omit<RisqTickAction, 'order' | 'intent' | 'execute'> {
  order?: RisqTickOrderFromServer;
  intent: RisqTickIntentFromServer;
  execute: RisqTickExecutionFromServer;
}

export declare interface RisqTickPlayerStateFromServer extends Omit<RisqTickPlayerState, 'researched_techs'> {
  researched_techs: Record<string, boolean>;
}

export declare interface RisqTickFrameFromServer {
  tick: number;
  terminal: boolean;
  phase: string;
  units: RisqTickUnitState[];
  buildings: RisqTickBuildingState[];
  resources: RisqTickResourceState[];
  players: RisqTickPlayerStateFromServer[];
  spaces: RisqTickSpaceState[];
  effects: RisqTickEffectFromServer[];
}

export declare interface RisqTurnJournalFromServer {
  schema_version: number;
  turn_number: number;
  tick_count: number;
  baseline: RisqTickFrameFromServer;
  frames: RisqTickFrameFromServer[];
  boundary_events: RisqTickFrameFromServer[];
  units: (RisqTickUnitState & { tick_actions: RisqTickActionFromServer[] })[];
  buildings: (RisqTickBuildingState & { tick_actions: RisqTickActionFromServer[] })[];
}
