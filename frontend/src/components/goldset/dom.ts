import { cardToImagePath } from '../game/util/card_util';
import type { GoldCard, GoldState } from './types';

export function element<K extends keyof HTMLElementTagNameMap>(tag: K, text: string = ''): HTMLElementTagNameMap[K] {
  const el = document.createElement(tag);
  el.textContent = text;
  return el;
}
export function cardName(card: Pick<GoldCard, 'number' | 'suit'>): string {
  const ranks = ['Jack', 'Queen', 'King', 'Ace'];
  return `${card.number >= 11 ? ranks[card.number - 11] : card.number} of ${['', 'hearts', 'clubs', 'diamonds', 'spades'][card.suit]}`;
}
export function cardImage(card: Pick<GoldCard, 'number' | 'suit'>): HTMLImageElement {
  const image = element('img');
  image.src = cardToImagePath(card);
  image.alt = cardName(card);
  image.title = image.alt;
  image.className = 'playing-card';
  return image;
}
export function actionName(state: GoldState, action: number): string {
  if (state.kind === 'bet') {
    return `Bid ${action}`;
  }
  const card = state.hand.find((c: GoldCard): boolean => c.action === action);
  return card ? `Play ${cardName(card)}` : 'Unknown card';
}
export function percent(value: number): string {
  return `${(100 * value).toFixed(1)}%`;
}
export function table(headers: string[], rows: string[][]): HTMLTableElement {
  const result = element('table');
  const head = element('tr');
  for (const label of headers) {
    head.append(element('th', label));
  }
  result.append(head);
  for (const row of rows) {
    const tr = element('tr');
    for (const value of row) {
      tr.append(element('td', value));
    }
    result.append(tr);
  }
  return result;
}
