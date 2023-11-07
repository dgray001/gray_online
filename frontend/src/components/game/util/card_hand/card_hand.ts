import {DwgElement} from '../../../dwg_element';
import {StandardCard, cardToIcon, cardToImagePath} from '../card_util';
import {until} from '../../../../scripts/util';

import html from './card_hand.html';

import './card_hand.scss';

export class DwgCardHand extends DwgElement {
  cards_container: HTMLDivElement;

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

  setCards(cards: StandardCard[]) {
    const card_els: HTMLDivElement[] = [];
    for (const card of cards) {
      const el = document.createElement('div');
      const img = document.createElement('img');
      img.src = cardToImagePath(card);
      img.draggable = false;
      img.alt = cardToIcon(card);
      el.appendChild(img);
      el.classList.add('card');
      const i = card_els.length;
      el.style.setProperty('--i', i.toString());
      card_els.push(el);
      el.addEventListener('click', () => {
        this.dispatchEvent(new CustomEvent<number>('play_card', {'detail': i}));
      });
    }
    this.cards_container.style.setProperty('--num-cards', card_els.length.toString());
    this.cards_container.replaceChildren(...card_els);
  }
}

customElements.define('dwg-card-hand', DwgCardHand);

declare global{
  interface HTMLElementTagNameMap {
    'dwg-card-hand': DwgCardHand;
  }
}
