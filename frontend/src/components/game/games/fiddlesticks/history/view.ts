import type { UpdateMessage } from '../../../data_models';
import type { StandardCard } from '../../../util/card_util';
import { cardToIcon, cardToName } from '../../../util/card_util';
import type { GameFiddlesticks, PlayCard, FiddlesticksPlayer, DealRound, PlayerBet } from '../fiddlesticks_data';
import { getSettings } from '../../../../../scripts/settings_store';
import { comparisonSummary, historyRounds } from './model';
import type { ComparisonSummary, HistoryRound, HistoryTrick } from './model';
import './history.scss';

function text(tag: string, value: string, class_name: string = ''): HTMLElement {
  const el = document.createElement(tag);
  el.className = class_name;
  el.textContent = value;
  return el;
}

function expander(title: string, parts: readonly Node[], class_name: string, open: boolean): HTMLDetailsElement {
  const details = document.createElement('details');
  details.className = class_name;
  details.open = open;
  const summary = document.createElement('summary');
  const content = text('span', '', 'history-summary');
  content.append(text('strong', title), ...parts);
  summary.append(content);
  details.append(summary);
  return details;
}

function playerName(game: GameFiddlesticks, id: number, viewer: number): string {
  return `${game.players[id].player.nickname}${id === viewer ? ' (you)' : ''}`;
}

function cardIcon(card: StandardCard): HTMLElement {
  const el = text('span', '', 'history-card');
  el.innerHTML = cardToIcon(card);
  el.setAttribute('role', 'img');
  el.setAttribute('aria-label', cardToName(card));
  return el;
}

export function renderFiddlesticksHistory(
  updates: readonly UpdateMessage[],
  game: GameFiddlesticks,
  viewer: number
): HTMLElement {
  const root = text('div', '', 'fiddlesticks-history');
  const rounds = historyRounds(updates, game.players.length, game.round_points, game.trick_points);
  root.append(text('h2', 'Round history'));
  const controls = text('div', '', 'history-controls');
  for (const open of [true, false]) {
    const button = document.createElement('button');
    button.type = 'button';
    button.textContent = open ? 'Expand all' : 'Collapse all';
    button.addEventListener('click', (): void => {
      root.querySelectorAll('details').forEach((details: HTMLDetailsElement): void => {
        details.open = open;
      });
    });
    controls.append(button);
  }
  root.append(controls);
  const summary = comparisonSummary(rounds);
  if (showComparisons() && summary.bets.total + summary.cards.total > 0) {
    root.append(summaryView(summary));
  }
  rounds
    .reverse()
    .forEach((round: HistoryRound, index: number): void => root.append(roundView(round, game, viewer, index === 0)));
  if (rounds.length === 0) {
    root.append(text('p', 'No rounds have been dealt yet.'));
  }
  return root;
}

function showComparisons(): boolean {
  return getSettings().fiddlesticks.show_ai_comparisons;
}

function summaryView(summary: ComparisonSummary): HTMLElement {
  const agreed = (counts: { total: number; matched: number }): string => `${counts.matched}/${counts.total}`;
  const total = {
    total: summary.bets.total + summary.cards.total,
    matched: summary.bets.matched + summary.cards.matched,
  };
  return text(
    'p',
    `Your AI comparison: ${agreed(total)} matched (bets ${agreed(summary.bets)}, cards ${agreed(summary.cards)})`,
    'history-caption'
  );
}

function roundView(round: HistoryRound, game: GameFiddlesticks, viewer: number, open: boolean): HTMLElement {
  const completed = round.tricks.filter((trick: HistoryTrick): boolean => !!trick.winner).length;
  const betting = round.bets.size < game.players.length;
  const total_bets = [...round.bets.values()].reduce((sum: number, bet: number): number => sum + bet, 0);
  const progress = betting
    ? `Betting ${round.bets.size}/${game.players.length}`
    : `${completed}/${round.deal.round} tricks`;
  const trump = text('span', 'Trump ', 'history-trump');
  trump.append(cardIcon(round.deal.trump));
  const details = expander(
    `Round ${round.deal.round}`,
    [trump, text('span', progress), text('span', `Bets ${total_bets}/${round.deal.round}`)],
    'history-round',
    open
  );
  const body = text('div', '', 'history-round-body');
  body.append(text('p', `Dealer: ${game.players[round.deal.dealer].player.nickname}`, 'history-caption'));
  body.append(resultsTable(round, game, viewer));
  const tricks = round.tricks.map(
    (trick: HistoryTrick, index: number): HTMLElement =>
      trickView(trick, index + 1, game, viewer, open && index === round.tricks.length - 1)
  );
  body.append(...tricks.reverse());
  if (round.tricks.length === 0) {
    body.append(text('p', 'Waiting for betting to finish.', 'history-caption'));
  }
  details.append(body);
  return details;
}

function resultsTable(round: HistoryRound, game: GameFiddlesticks, viewer: number): HTMLElement {
  const table = text('table', '', 'history-results');
  table.setAttribute('aria-label', `Round ${round.deal.round} bets and results`);
  const heading = text('tr', '');
  for (const label of ['Player', 'Bet', 'Won', 'Points', 'Total']) {
    const cell = text('th', label);
    cell.setAttribute('scope', 'col');
    heading.append(cell);
  }
  const head = text('thead', '');
  head.append(heading);
  table.append(head);
  const body = text('tbody', '');
  game.players.forEach((_player: FiddlesticksPlayer, id: number): void =>
    body.append(resultRow(round, game, viewer, id))
  );
  table.append(body);
  const scroll = text('div', '', 'history-results-scroll');
  scroll.setAttribute('role', 'region');
  scroll.setAttribute('aria-label', `Round ${round.deal.round} scores`);
  scroll.tabIndex = 0;
  scroll.append(table);
  const section = text('div', '', 'history-results-section');
  section.append(
    text('h3', 'Bets and results', 'history-results-title'),
    scroll,
    text('p', 'Scroll sideways for all scores.', 'history-scroll-hint')
  );
  return section;
}

function comparisonTag(matched: boolean, detail: string): HTMLElement {
  return text('span', matched ? 'Match' : detail, matched ? 'history-tag history-winner' : 'history-tag');
}

function resultRow(round: HistoryRound, game: GameFiddlesticks, viewer: number, id: number): HTMLElement {
  const { bet, won, points, total } = round.scores[id];
  const row = text('tr', '', id === viewer ? 'history-you' : '');
  const name = text('th', playerName(game, id, viewer));
  name.setAttribute('scope', 'row');
  const comparison = round.bet_comparisons.get(id);
  const bet_cell = text('td', bet?.toString() ?? '-');
  if (comparison && showComparisons()) {
    bet_cell.append(comparisonTag(comparison.matched, `Model ${comparison.model_id}: ${comparison.ai_choice}`));
  }
  row.append(
    name,
    bet_cell,
    text('td', won.toString()),
    text('td', points === undefined ? '-' : `+${points}`),
    text('td', total.toString())
  );
  if (points !== undefined && bet === won) {
    row.classList.add('history-exact');
  }
  return row;
}

function trickView(
  trick: HistoryTrick,
  number: number,
  game: GameFiddlesticks,
  viewer: number,
  open: boolean
): HTMLElement {
  const last = trick.plays[trick.plays.length - 1];
  const parts = trick.winner
    ? [text('span', `${playerName(game, trick.winner.player_id, viewer)} won`), cardIcon(trick.winner.card)]
    : [
        text('span', `${trick.plays.length}/${game.players.length} played`),
        text('span', `${playerName(game, (last.player_id + 1) % game.players.length, viewer)} next`),
      ];
  const details = expander(`Trick ${number}`, parts, 'history-trick', open);
  const list = text('ol', '', 'history-plays');
  trick.plays.forEach((play: PlayCard, index: number): void => {
    const row = text('li', '', play.player_id === viewer ? 'history-you' : '');
    row.append(text('span', `${index + 1}. ${playerName(game, play.player_id, viewer)}`), cardIcon(play.card));
    if (index === 0) {
      row.append(text('span', 'Led', 'history-tag'));
    }
    if (play.ai_comparison && showComparisons()) {
      const { matched, model_id, ai_choice } = play.ai_comparison;
      row.append(comparisonTag(matched, `Model ${model_id}: ${cardToName(ai_choice)}`));
    }
    if (play === trick.winner) {
      row.append(text('span', 'Winner', 'history-tag history-winner'));
    }
    list.append(row);
  });
  details.append(list);
  return details;
}

export function renderFiddlesticksUpdate(update: UpdateMessage, game: GameFiddlesticks): HTMLElement {
  const root = text('div', '');
  const row = (...parts: (string | Node)[]): void => {
    const el = text('div', '');
    el.append(...parts);
    root.append(el);
  };
  switch (update.kind) {
    case 'deal-round': {
      const deal = update.content as DealRound;
      row(`Round ${deal.round} dealt by ${game.players[deal.dealer].player.nickname}`);
      row('Card flipped over as trump was ', cardIcon(deal.trump));
      if (deal.cards) {
        row('Cards dealt to you: ', ...deal.cards.map(cardIcon));
      }
      break;
    }
    case 'bet': {
      const bet = update.content as PlayerBet;
      row(`${game.players[bet.player_id].player.nickname} bet ${bet.amount} points`);
      break;
    }
    case 'play-card': {
      const play = update.content as PlayCard;
      row(`${game.players[play.player_id].player.nickname} played `, cardIcon(play.card));
      break;
    }
    default:
      row(`Unknown update type for fiddlesticks: ${update.kind}`);
  }
  return root;
}
