import type { Point2D } from '../../../../util/objects2d';
import type { ReplayEvent, ReplayProduction } from '../../model/replay_event';
import type { RisqTickLocation } from '../../model/tick_actions';
import type { RisqTurnJournal } from '../../model/turn_journal';
import type { GameRisq, RisqSpace } from '../../model/types';
import { RisqProducibleKind } from '../../model/types';
import type { RisqTurnReport } from '../../transport/turn_report';
import { unitImage } from '../../rendering/assets/unit';
import { buildingImage, techImage } from '../../rendering/assets/buildings';
import { hiredEvents as groupedHires } from './hired_events';

type ProductionRow = ReplayProduction & { count: number; location?: RisqTickLocation };

function boardCenter(game: GameRisq): Point2D {
  const spaces = game.spaces.flat().filter((space: RisqSpace | undefined): space is RisqSpace => !!space);
  return {
    x: spaces.reduce((sum: number, space: RisqSpace): number => sum + space.coordinate.x, 0) / spaces.length,
    y: spaces.reduce((sum: number, space: RisqSpace): number => sum + space.coordinate.y, 0) / spaces.length,
  };
}

export function productionReportEvents(
  events: ReplayEvent[],
  journal: RisqTurnJournal,
  game: GameRisq,
  player_id: number
): ReplayEvent[] {
  const report = game.players[player_id]?.turn_report;
  if (report?.turn !== journal.turn_number) {
    return events;
  }
  const available = events.filter((event: ReplayEvent): boolean => event.player_id === player_id);
  const grouped = productionRows(report, game).map((row: ProductionRow, index: number): ReplayEvent => {
    const matches = available.filter(
      (event: ReplayEvent): boolean =>
        event.production?.kind === row.kind &&
        event.production.item_id === row.item_id &&
        event.production.source === row.source &&
        (!row.location || (!!event.locations[0] && locationKey(event.locations[0]) === locationKey(row.location)))
    );
    if (row.source === 'built') {
      matches.splice(1);
    }
    for (const event of matches) {
      available.splice(available.indexOf(event), 1);
    }
    return rowEvent(row, index, matches, journal, game, player_id);
  });
  const hires = groupedHires(journal, game, player_id).map(
    (event: ReplayEvent, index: number): ReplayEvent => ({
      ...event,
      production: { ...event.production!, row: grouped.length + index },
    })
  );
  return [...events.filter((event: ReplayEvent): boolean => event.player_id !== player_id), ...grouped, ...hires];
}

function locationKey(location: RisqTickLocation): string {
  return `${location.space.x},${location.space.y},${location.zone.x},${location.zone.y}`;
}

function rowEvent(
  row: ProductionRow,
  index: number,
  matches: ReplayEvent[],
  journal: RisqTurnJournal,
  game: GameRisq,
  player_id: number
): ReplayEvent {
  const first = matches[0];
  const locations = [
    ...new Map(
      matches
        .flatMap((event: ReplayEvent): RisqTickLocation[] => event.locations)
        .map((location: RisqTickLocation): [string, RisqTickLocation] => [locationKey(location), location])
    ).values(),
  ];
  if (locations.length === 0 && row.location) {
    locations.push(row.location);
  }
  const icon =
    row.kind === RisqProducibleKind.UNIT
      ? unitImage(row.item_id, true)
      : row.kind === RisqProducibleKind.BUILDING
        ? buildingImage(row.item_id, false, true)
        : techImage(row.item_id);
  return {
    kind: 'production',
    player_id,
    icon: first?.icon ?? icon,
    title: `${row.count > 1 ? `${row.count}× ` : ''}${first?.title ?? `${row.source} ${row.item_id}`}`,
    tick:
      matches.length > 0 ? Math.max(...matches.map((event: ReplayEvent): number => event.tick)) : journal.tick_count,
    locations,
    production: { kind: row.kind, item_id: row.item_id, source: row.source, row: index },
    end_of_turn: row.source === 'hired' || matches.length === 0,
    map_position: locations.length === 0 ? boardCenter(game) : undefined,
  };
}

function productionRows(report: RisqTurnReport, game: GameRisq): ProductionRow[] {
  const rows: ProductionRow[] = [];
  for (const unit of report.production.units_created) {
    rows.push({ kind: RisqProducibleKind.UNIT, item_id: unit.unit_id, source: 'trained', count: unit.count });
  }
  for (const building of game.has_buildings !== false ? report.production.buildings_built : []) {
    rows.push({
      kind: RisqProducibleKind.BUILDING,
      item_id: building.building_id,
      source: 'built',
      count: 1,
      location: { space: building.space, zone: building.zone },
    });
  }
  for (const id of report.production.techs_researched) {
    rows.push({ kind: RisqProducibleKind.TECH, item_id: id, source: 'researched', count: 1 });
  }
  return rows;
}
