import type { GoldState } from './types';
import { cardImage, element, percent, table } from './dom';

export function renderState(host: HTMLElement, state: GoldState): void {
  host.replaceChildren();
  host.append(element('h2', state.kind === 'bet' ? 'Choose your bid' : 'Choose your card'));
  host.append(
    element('p', `You are player ${state.seat + 1}. Dealer: player ${state.dealer + 1}. Goal: win the game.`)
  );
  host.append(
    element(
      'p',
      `Round ${state.progress.index + 1} of ${state.progress.total} · ${state.hand_size} cards · ${state.increasing ? 'ascending' : 'descending'} · ${state.progress.remaining} rounds after this one`
    )
  );
  host.append(
    element(
      'p',
      `Exact-bid points: ${state.round_points} + ${state.trick_points} × bid. Maximum points available after this hand: ${state.progress.future_points}.`
    )
  );
  const trump = element('div');
  trump.className = 'trump';
  trump.append(element('strong', 'Trump'), cardImage(state.trump));
  host.append(trump);
  if (state.kind === 'bet') {
    const hand = element('div');
    hand.className = 'choices';
    for (const card of state.hand) {
      if (!card.played) {
        hand.append(cardImage(card));
      }
    }
    host.append(element('h3', 'Your hand'), hand);
  }
  const rows = state.players.map((p): string[] => [
    `${p.seat + 1}${p.seat === state.seat ? ' (you)' : ''}`,
    `${p.score}`,
    p.has_bid ? `${p.bid}` : '—',
    `${p.tricks}`,
    p.voids
      .map((v: boolean, i: number): string => (v ? ['Heart', 'Club', 'Diamond', 'Spade'][i] : ''))
      .filter(Boolean)
      .join(', ') || 'unknown',
    p.profile?.hands
      ? `${percent(p.profile.mean_bid)} / ${percent(p.profile.exact_rate)} (${p.profile.hands} hands)`
      : '—',
  ]);
  host.append(table(['Player', 'Points', 'Bid', 'Tricks', 'Known voids', 'Bid/hand · exact rate'], rows));
  if (state.kind === 'bet') {
    host.append(element('p', `First trick leader: player ${((state.dealer + 1) % state.players.length) + 1}.`));
    return;
  }
  host.append(element('h3', 'Current trick'));
  const trick = element('div');
  trick.className = 'trick';
  for (const [index, card] of (state.trick ?? []).entries()) {
    const box = element('div');
    box.append(element('span', `Player ${((state.trick_leader + index) % state.players.length) + 1}`), cardImage(card));
    trick.append(box);
  }
  if (!trick.children.length) {
    trick.append(
      element(
        'p',
        state.kind === 'play-card'
          ? 'You lead this trick.'
          : `First trick leader: player ${((state.dealer + 1) % state.players.length) + 1}.`
      )
    );
  }
  host.append(trick);
}
