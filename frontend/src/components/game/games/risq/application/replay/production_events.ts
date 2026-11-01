import type { GameRisq, RisqProducible, RisqBuilding } from '../../model/types';
import { RisqProducibleKind } from '../../model/types';
import type { RisqTickAction, RisqTickLocation } from '../../model/tick_actions';
import type { RisqJournalBuilding, RisqJournalUnit, RisqTickFrame, RisqTurnJournal } from '../../model/turn_journal';
import { buildingImage, techImage } from '../../rendering/assets/buildings';
import { unitImage } from '../../rendering/assets/unit';
import type { ReplayEvent } from '../../model/replay_event';

export interface ProductionEvent extends ReplayEvent {
  kind: 'production';
}

interface ProductionAction {
  actor: RisqJournalBuilding | RisqJournalUnit;
  action: RisqTickAction;
}

function executedActions(journal: RisqTurnJournal): Map<number, ProductionAction[]> {
  const actions = new Map<number, ProductionAction[]>();
  for (const actor of [...journal.units.values(), ...journal.buildings.values()]) {
    for (const action of actor.tick_actions) {
      if (action.execute.outcome !== 'executed') {
        continue;
      }
      if (action.execute.kind !== 'production' && action.execute.kind !== 'build') {
        continue;
      }
      const at_tick = actions.get(action.tick) ?? [];
      at_tick.push({ actor, action });
      actions.set(action.tick, at_tick);
    }
  }
  return actions;
}

function distinctLocations(locations: RisqTickLocation[]): RisqTickLocation[] {
  return [
    ...new Map(
      locations.map((location: RisqTickLocation): [string, RisqTickLocation] => [
        `${location.space.x},${location.space.y},${location.zone.x},${location.zone.y}`,
        location,
      ])
    ).values(),
  ];
}

export function productionEvents(journal: RisqTurnJournal, game: GameRisq): ProductionEvent[] {
  const events: ProductionEvent[] = [];
  const units = new Map(journal.baseline.units);
  const buildings = new Map(journal.baseline.buildings);
  const players = new Map(journal.baseline.players);
  const actions = executedActions(journal);
  for (const frame of journal.frames.filter((frame: RisqTickFrame): boolean => !frame.terminal)) {
    const executed = actions.get(frame.tick) ?? [];
    for (const unit of frame.units.values()) {
      const producer = executed.find(
        ({ actor, action }: ProductionAction): boolean =>
          actor.player_id === unit.player_id &&
          action.intent.producible_kind === RisqProducibleKind.UNIT &&
          action.intent.item_id === unit.unit_id &&
          (unit.garrisoned_in === actor.internal_id ||
            (action.intent.location.space.x === unit.space_coordinate.x &&
              action.intent.location.space.y === unit.space_coordinate.y &&
              action.intent.location.zone.x === unit.zone_coordinate.x &&
              action.intent.location.zone.y === unit.zone_coordinate.y))
      );
      if (!units.has(unit.internal_id) && producer) {
        events.push({
          kind: 'production',
          tick: frame.tick,
          player_id: unit.player_id,
          title: `${unit.display_name} created`,
          icon: unitImage(unit.unit_id, true),
          locations: [producer.action.intent.location],
        });
      }
      units.set(unit.internal_id, unit);
    }
    for (const building of frame.buildings.values()) {
      const builders = executed.filter(
        ({ actor, action }: ProductionAction): boolean =>
          actor.player_id === building.player_id &&
          action.execute.kind === 'build' &&
          action.execute.target?.kind === 'building' &&
          action.execute.target.internal_id === building.internal_id
      );
      const previous = buildings.get(building.internal_id);
      if (!building.under_construction && builders.length > 0 && (!previous || previous.under_construction)) {
        events.push({
          kind: 'production',
          tick: frame.tick,
          player_id: building.player_id,
          title: `${building.display_name} completed`,
          icon: buildingImage(building.building_id, false, true),
          locations: distinctLocations([
            { space: building.space_coordinate, zone: building.zone_coordinate },
            ...builders.map(({ action }: ProductionAction): RisqTickLocation => action.intent.location),
          ]),
        });
      }
      buildings.set(building.internal_id, building);
    }
    for (const player of frame.players.values()) {
      for (const [id, researched] of player.researched_techs) {
        const researchers = executed.filter(
          ({ actor, action }: ProductionAction): boolean =>
            actor.player_id === player.player_id &&
            action.intent.producible_kind === RisqProducibleKind.TECH &&
            action.intent.item_id === id
        );
        if (researched && !players.get(player.player_id)?.researched_techs.get(id) && researchers.length > 0) {
          const tech = [...game.buildings.values()]
            .flatMap((building: RisqBuilding): RisqProducible[] => building.produces)
            .find((item: RisqProducible): boolean => item.kind === RisqProducibleKind.TECH && item.id === id);
          events.push({
            kind: 'production',
            tick: frame.tick,
            player_id: player.player_id,
            title: `${tech?.display_name ?? `Technology ${id}`} researched`,
            icon: techImage(id),
            locations: distinctLocations(
              researchers.map(({ action }: ProductionAction): RisqTickLocation => action.intent.location)
            ),
          });
        }
      }
      players.set(player.player_id, player);
    }
  }
  return events;
}
