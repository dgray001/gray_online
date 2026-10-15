import type { RisqTickAction, RisqTickEffect, RisqTickOrder, RisqTickTarget } from '../model/tick_actions';
import type { RisqTickFrame, RisqTickPlayerState, RisqTurnJournal } from '../model/turn_journal';
import type {
  RisqTickActionFromServer,
  RisqTickEffectFromServer,
  RisqTickFrameFromServer,
  RisqTickOrderFromServer,
  RisqTickPlayerStateFromServer,
  RisqTickTargetFromServer,
  RisqTurnJournalFromServer,
} from './turn_journal_types';

function serverToTickTarget(target?: RisqTickTargetFromServer): RisqTickTarget | undefined {
  return target?.kind === undefined ? undefined : (target as RisqTickTarget);
}

function serverToTickOrder(order?: RisqTickOrderFromServer): RisqTickOrder | undefined {
  return order && { ...order, source_actor: serverToTickTarget(order.source_actor) };
}

function serverToTickEffect(effect: RisqTickEffectFromServer): RisqTickEffect {
  return {
    ...effect,
    order: serverToTickOrder(effect.order),
    actor: serverToTickTarget(effect.actor),
    target: serverToTickTarget(effect.target),
  };
}

export function serverToTickAction(action: RisqTickActionFromServer): RisqTickAction {
  const { food_spent: food, wood_spent: wood, stone_spent: stone, gold_spent: gold, ...execute } = action.execute;
  return {
    ...action,
    order: serverToTickOrder(action.order),
    intent: { ...action.intent, target: serverToTickTarget(action.intent.target) },
    execute: {
      ...execute,
      cost: { food, wood, stone, gold },
      target: serverToTickTarget(execute.target),
      effects: execute.effects.map(serverToTickEffect),
    },
  };
}

function indexBy<T>(items: T[], key: (item: T) => number): Map<number, T> {
  return new Map(items.map((item): [number, T] => [key(item), item]));
}

function serverToTickPlayer(player: RisqTickPlayerStateFromServer): RisqTickPlayerState {
  const techs = Object.entries(player.researched_techs).map(([id, done]): [number, boolean] => [Number(id), done]);
  return { ...player, researched_techs: new Map(techs) };
}

function serverToTickFrame(frame: RisqTickFrameFromServer): RisqTickFrame {
  return {
    ...frame,
    units: indexBy(frame.units, (state) => state.internal_id),
    buildings: indexBy(frame.buildings, (state) => state.internal_id),
    resources: indexBy(frame.resources, (state) => state.internal_id),
    players: indexBy(frame.players.map(serverToTickPlayer), (state) => state.player_id),
    spaces: indexBy(frame.spaces, (state) => state.coordinate_key),
    effects: frame.effects.map(serverToTickEffect),
  };
}

export function serverToTurnJournal(journal?: RisqTurnJournalFromServer | null): RisqTurnJournal | undefined {
  if (!journal) {
    return undefined;
  }
  const with_actions = <T extends { tick_actions: RisqTickActionFromServer[] }>(
    state: T
  ): Omit<T, 'tick_actions'> & { tick_actions: RisqTickAction[] } => ({
    ...state,
    tick_actions: state.tick_actions.map(serverToTickAction),
  });
  return {
    ...journal,
    baseline: serverToTickFrame(journal.baseline),
    frames: journal.frames.map(serverToTickFrame),
    boundary_events: journal.boundary_events.map(serverToTickFrame),
    units: indexBy(journal.units.map(with_actions), (unit) => unit.internal_id),
    buildings: indexBy(journal.buildings.map(with_actions), (building) => building.internal_id),
  };
}
