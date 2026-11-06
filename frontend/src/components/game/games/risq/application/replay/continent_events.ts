import type { ReplayEvent } from '../../model/replay_event';
import type { GameRisq, RisqRegion } from '../../model/types';
import type { RisqTurnJournal } from '../../model/turn_journal';
import { invertPair } from '../../model/coordinates';

type Change = 'gained' | 'lost';

export function continentEvents(journal: RisqTurnJournal, game: GameRisq, player_id: number): ReplayEvent[] {
  const events = new Map<string, ReplayEvent>();
  const add = (region: RisqRegion, owner: number, change: Change): void => {
    events.set(`${region.name}/${owner}/${change}`, continentEvent(region, owner, change, journal));
  };
  for (const region of game.regions) {
    const previous = journal.region_owners?.[region.name];
    if (previous === undefined || previous === region.owner) {
      continue;
    }
    if (previous >= 0) {
      add(region, previous, 'lost');
    }
    if (region.owner >= 0) {
      add(region, region.owner, 'gained');
    }
  }
  const report = game.players[player_id]?.turn_report;
  if (report?.turn === journal.turn_number) {
    for (const row of report.regions) {
      if (row.held_start === row.held_end) {
        continue;
      }
      const region = game.regions.find((area: RisqRegion): boolean => area.name === row.name);
      if (region) {
        add(region, player_id, row.held_end ? 'gained' : 'lost');
      }
    }
  }
  return [...events.values()];
}

function continentEvent(region: RisqRegion, player_id: number, change: Change, journal: RisqTurnJournal): ReplayEvent {
  return {
    kind: 'continent',
    player_id,
    tick: journal.tick_count,
    end_of_turn: true,
    title: `${region.name} ${change}`,
    icon: 'icons/crown64',
    continent: { name: region.name, change, gold_bonus: region.gold_bonus },
    locations: region.spaces.map((key: number): ReplayEvent['locations'][number] => ({
      space: invertPair(key),
      zone: { x: 0, y: 0 },
    })),
  };
}
