import {DwgElement} from '../../../dwg_element';
import {StandardCard} from '../card_util';
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
  }

  setCards(cards: StandardCard[]) {
    const card_els: HTMLDivElement[] = [];
    for (const card of cards) {
      const el = document.createElement('div');
      const img = document.createElement('img');
      // f8dusapf8wauh fuawhfn uwaehhfpsfuadsf sa*************************************************************
      el.classList.add('card');
      card_els.push(el);
    }
    this.cards_container.replaceChildren(...card_els);
  }
}

customElements.define('dwg-card-hand', DwgCardHand);

declare global{
  interface HTMLElementTagNameMap {
    'dwg-card-hand': DwgCardHand;
  }
}
