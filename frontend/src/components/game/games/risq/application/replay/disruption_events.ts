import type { Point2D } from '../../../../util/objects2d';
import type { ReplayEvent } from '../../model/replay_event';
import type { RisqTickLocation } from '../../model/tick_actions';
import type { RisqTickBuildingState, RisqTickUnitState, RisqTurnJournal } from '../../model/turn_journal';
import type { GameRisq, RisqSpace } from '../../model/types';
import { RisqOrderType } from '../../model/types';
import { invertPair, invertZoneKey } from '../../model/coordinates';
import type { RisqReportOrderFailure } from '../../transport/turn_report';
import { shortLabel } from '../../canvas_components/order_row/order_row_resolve';

type Receipt = { failure: RisqReportOrderFailure; tick: number; location?: RisqTickLocation };
type Subject = RisqTickBuildingState | RisqTickUnitState;

function boardCenter(game: GameRisq): Point2D {
  const spaces = game.spaces.flat().filter((space: RisqSpace | undefined): space is RisqSpace => !!space);
  return {
    x: spaces.reduce((sum: number, space: RisqSpace): number => sum + space.coordinate.x, 0) / spaces.length,
    y: spaces.reduce((sum: number, space: RisqSpace): number => sum + space.coordinate.y, 0) / spaces.length,
  };
}

function takeReceipt(receipts: Receipt[], failure: RisqReportOrderFailure): Receipt | undefined {
  const index = receipts.findIndex(
    (receipt: Receipt): boolean =>
      receipt.failure.order_type === failure.order_type &&
      receipt.failure.target_id === failure.target_id &&
      receipt.failure.reason === failure.reason
  );
  return index < 0 ? undefined : receipts.splice(index, 1)[0];
}

function rejectionReceipts(journal: RisqTurnJournal, player_id: number): Receipt[] {
  const receipts: Receipt[] = [];
  const subjects = new Map<string, Subject>();
  for (const frame of [journal.baseline, ...journal.frames, ...journal.boundary_events]) {
    for (const unit of frame.units.values()) {subjects.set(`unit:${unit.internal_id}`, unit);}
    for (const building of frame.buildings.values()) {subjects.set(`building:${building.internal_id}`, building);}
    for (const effect of frame.effects) {
      if (effect.kind !== 'order_rejected' || !effect.order || effect.actor?.player_id !== player_id) {continue;}
      const subject = subjects.get(`${effect.actor.kind}:${effect.actor.internal_id}`);
      receipts.push({
        failure: { ...effect.order, reason: effect.reason ?? '' },
        tick: effect.tick,
        location: subject ? { space: subject.space_coordinate, zone: subject.zone_coordinate } : undefined,
      });
    }
  }
  return receipts;
}

export function disruptionEvents(journal: RisqTurnJournal, game: GameRisq, player_id: number): ReplayEvent[] {
  const report = game.players[player_id]?.turn_report;
  if (report?.turn !== journal.turn_number) {return [];}
  const receipts = rejectionReceipts(journal, player_id);
  return report.orders.failures.map((failure: RisqReportOrderFailure): ReplayEvent => {
    const receipt = takeReceipt(receipts, failure);
    const location =
      failure.order_type === RisqOrderType.OrderType_BuyMercenary
        ? invertZoneKey(invertPair(failure.target_id).y)
        : receipt?.location;
    return {
      kind: 'disruption',
      player_id,
      title: shortLabel(failure.order_type),
      reason: failure.reason,
      icon: 'icons/interrupt_current128',
      tick: Math.min(receipt?.tick ?? journal.tick_count, journal.tick_count),
      end_of_turn: !receipt || receipt.tick > journal.tick_count,
      locations: location ? [location] : [],
      map_position: location ? undefined : boardCenter(game),
    };
  });
}
