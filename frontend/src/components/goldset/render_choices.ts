import type { GoldCard, GoldItem } from './types';
import { cardImage, cardName, element } from './dom';

export function renderChoices(host: HTMLElement, item: GoldItem, choose: (action: number) => void): void {
  host.replaceChildren();
  host.append(element('h3', item.state.kind === 'bet' ? 'Your answer' : 'Your hand'));
  const choices = element('div');
  choices.className = 'choices';
  if (item.state.kind === 'bet') {
    for (const action of item.legal) {
      const button = element(
        'button',
        `${action} · ${item.state.round_points + item.state.trick_points * action} points`
      );
      button.id = `choice-${action}`;
      button.dataset.action = `${action}`;
      button.type = 'button';
      button.onclick = (): void => choose(action);
      choices.append(button);
    }
  } else {
    const hand = item.state.hand
      .filter((c: GoldCard): boolean => !c.played)
      .sort((a: GoldCard, b: GoldCard): number => a.suit - b.suit || a.number - b.number);
    for (const card of hand) {
      const button = element('button');
      button.id = `choice-${card.action}`;
      button.dataset.action = `${card.action}`;
      button.type = 'button';
      button.disabled = !item.legal.includes(card.action);
      button.setAttribute('aria-label', cardName(card));
      button.append(cardImage(card));
      button.onclick = (): void => choose(card.action);
      choices.append(button);
    }
  }
  host.append(choices);
}
