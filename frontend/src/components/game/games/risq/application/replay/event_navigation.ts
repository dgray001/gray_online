import type { ReplayEvent } from '../../model/replay_event';

export function replayEventStop(event: ReplayEvent): number {
  return event.kind === 'battle' ? event.tick - 0.5 : event.tick;
}

export function adjacentReplayEvent(
  events: readonly ReplayEvent[],
  selected: ReplayEvent | undefined,
  stop: number,
  direction: -1 | 1
): ReplayEvent | undefined {
  const ordered = [...events].sort(
    (a: ReplayEvent, b: ReplayEvent): number =>
      replayEventStop(a) - replayEventStop(b) || Number(!!a.end_of_turn) - Number(!!b.end_of_turn)
  );
  const index = selected && replayEventStop(selected) === stop ? ordered.indexOf(selected) : -1;
  if (index >= 0) {
    return ordered[index + direction];
  }
  return direction === 1
    ? ordered.find((event: ReplayEvent): boolean => replayEventStop(event) >= stop)
    : ordered.reverse().find((event: ReplayEvent): boolean => replayEventStop(event) <= stop);
}
