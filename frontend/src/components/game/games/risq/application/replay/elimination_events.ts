import type { Point2D } from '../../../../util/objects2d';
import type { ReplayEvent } from '../../model/replay_event';
import type { RisqTickLocation } from '../../model/tick_actions';
import type { RisqTickBuildingState, RisqTickUnitState, RisqTurnJournal } from '../../model/turn_journal';
import type { GameRisq, RisqSpace } from '../../model/types';
import { RisqVisibilityLevel } from '../../model/types';

type LocatedEntity = RisqTickBuildingState | RisqTickUnitState;

function centroid(spaces: RisqSpace[]): Point2D | undefined {
  if (spaces.length === 0) {
    return undefined;
  }
  return {
    x: spaces.reduce((sum: number, space: RisqSpace): number => sum + space.coordinate.x, 0) / spaces.length,
    y: spaces.reduce((sum: number, space: RisqSpace): number => sum + space.coordinate.y, 0) / spaces.length,
  };
}

function unknownPosition(game: GameRisq): Point2D | undefined {
  const spaces = game.spaces.flat().filter((space: RisqSpace | undefined): space is RisqSpace => !!space);
  return (
    centroid(spaces.filter((space: RisqSpace): boolean => space.visibility === RisqVisibilityLevel.UNEXPLORED)) ??
    centroid(spaces.filter((space: RisqSpace): boolean => space.visibility === RisqVisibilityLevel.FOG)) ??
    centroid(spaces)
  );
}

function lastLocations(journal: RisqTurnJournal, player_id: number): RisqTickLocation[] {
  for (const frame of [journal.baseline, ...journal.frames, ...journal.boundary_events].reverse()) {
    const entities = [...frame.units.values(), ...frame.buildings.values()].filter(
      (entity: LocatedEntity): boolean => entity.player_id === player_id
    );
    if (entities.length > 0) {
      return [
        ...new Map(
          entities.map((entity: LocatedEntity): [string, RisqTickLocation] => [
            `${entity.space_coordinate.x},${entity.space_coordinate.y},${entity.zone_coordinate.x},${entity.zone_coordinate.y}`,
            { space: entity.space_coordinate, zone: entity.zone_coordinate },
          ])
        ).values(),
      ];
    }
  }
  return [];
}

function eliminatedPlayers(journal: RisqTurnJournal, game: GameRisq, player_id: number): Set<number> {
  const report = game.players[player_id]?.turn_report;
  const eliminated = new Set<number>();
  if (report?.turn === journal.turn_number) {
    for (const score of report.scores) {
      if (score.eliminated_this_turn) {
        eliminated.add(score.player_id);
      }
    }
  }
  const players = new Map(journal.baseline.players);
  for (const frame of [...journal.frames, ...journal.boundary_events]) {
    for (const player of frame.players.values()) {
      if (player.eliminated && players.get(player.player_id)?.eliminated === false) {
        eliminated.add(player.player_id);
      }
      players.set(player.player_id, player);
    }
  }
  return eliminated;
}

export function eliminationEvents(journal: RisqTurnJournal, game: GameRisq, player_id: number): ReplayEvent[] {
  return [...eliminatedPlayers(journal, game, player_id)].map((id: number): ReplayEvent => {
    const locations = lastLocations(journal, id);
    return {
      kind: 'eliminated',
      tick: journal.tick_count,
      player_id: id,
      title: 'Player eliminated',
      icon: 'icons/close64',
      locations,
      map_position: locations.length === 0 ? unknownPosition(game) : undefined,
    };
  });
}
