import type { GoldState } from './types';
import { cardName, element, table } from './dom';

export function renderHistory(host: HTMLElement, state: GoldState): void {
  host.replaceChildren();
  const details = element('details');
  details.append(element('summary', `Public history (${state.history?.length ?? 0} actions)`));
  const rows = (state.history ?? []).map((e): string[] => [
    `${e.round + 1} (${e.hand} cards)`,
    e.kind === 'bet' ? 'Bidding' : `${e.trick} / ${e.order + 1}`,
    `Player ${e.seat + 1}`,
    e.kind === 'bet' ? `Bid ${e.bid}` : cardName(e),
  ]);
  details.append(table(['Round', 'Trick / order', 'Player', 'Action'], rows));
  host.append(details);
}
