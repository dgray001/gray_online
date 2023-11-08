import {DwgElement} from '../../../dwg_element';
import {StandardCard, cardToIcon, cardToImagePath} from '../card_util';
import {until} from '../../../../scripts/util';

import html from './card_hand.html';

import './card_hand.scss';

export class DwgCardHand extends DwgElement {
  cards_container: HTMLDivElement;
  cards = new Map<number, HTMLDivElement>();

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('cards_container');
  }

  protected override async parsedCallback(): Promise<void> {
    await until(() => !!this.clientHeight);
    this.style.setProperty('--height', `${this.clientHeight.toString()}px`);
    this.style.setProperty('--width', `${this.clientWidth.toString()}px`);
  }

  setCards(cards: StandardCard[], cards_played: number[] = []) {
    const card_els: HTMLDivElement[] = [];
    let cards_skipped = 0;
    for (const [i, card] of cards.entries()) {
      if (cards_played.some(index => index === i)) {
        cards_skipped++;
        continue;
      }
      const el = document.createElement('div');
      const img = document.createElement('img');
      img.src = cardToImagePath(card);
      img.draggable = false;
      img.alt = cardToIcon(card);
      el.appendChild(img);
      el.classList.add('card');
      el.style.setProperty('--i', (i - cards_skipped).toString());
      card_els.push(el);
      this.cards.set(i, el);
      el.addEventListener('click', () => {
        this.dispatchEvent(new CustomEvent<number>('play_card', {'detail': i}));
      });
    }
    this.cards_container.style.setProperty('--num-cards', this.cards.size.toString());
    this.cards_container.replaceChildren(...card_els);
  }

  playCard(index: number) {
    const card_el = this.cards.get(index);
    if (!card_el) {
      console.error('Trying to play card that does not exist');
      return;
    }
    card_el.remove();
    this.cards.delete(index);
    this.cards_container.style.setProperty('--num-cards', this.cards.size.toString());
    for (const card of this.cards.values()) {
      const i = parseInt(card.style.getPropertyValue('--i')) ?? NaN;
      if (isNaN(i)) {
        console.error('Found NaN for --i of card el where integer should be')
        continue;
      }
      if (i <= index) {
        continue;
      }
      card.style.setProperty('--i', (i - 1).toString());
    }
  }
}

customElements.define('dwg-card-hand', DwgCardHand);

declare global{
  interface HTMLElementTagNameMap {
    'dwg-card-hand': DwgCardHand;
  }
}
