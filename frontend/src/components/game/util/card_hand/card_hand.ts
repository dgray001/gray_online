import {DwgElement} from '../../../dwg_element';
import {StandardCard, cardToIcon, cardToImagePath} from '../card_util';
import {createLock, until} from '../../../../scripts/util';
import {Point2D} from '../objects2d';

import html from './card_hand.html';

import './card_hand.scss';

interface CardData {
  i: number; // actual card index in hand
  i_dom: number; // displayed card index
  el: HTMLDivElement;
}

interface CardDraggingData {
  dragging: boolean;
  index: number;
  touch_identifier: number;
  start: Point2D;
  hovering_play_drop: boolean;
}

export class DwgCardHand extends DwgElement {
  cards_container: HTMLDivElement;
  play_drop: HTMLDivElement;
  cancel_drop: HTMLDivElement;

  play_drop_cutoff = 0; // if y is less than this then card is being played
  cards = new Map<number, CardData>();
  dragging_data: CardDraggingData = {
    dragging: false,
    index: -1,
    touch_identifier: -1,
    start: {x: 0, y: 0},
    hovering_play_drop: false,
  };
  dragging_lock = createLock();

  constructor() {
    super();
    this.classList.add('hidden');
    this.htmlString = html;
    this.configureElement('cards_container');
    this.configureElement('play_drop');
    this.configureElement('cancel_drop');
  }

  protected override async parsedCallback(): Promise<void> {
    await until(() => !!this.clientHeight);
    const rect = this.getBoundingClientRect();
    this.style.setProperty('--height', `${rect.height.toString()}px`);
    this.style.setProperty('--width', `${rect.width.toString()}px`);
    const play_drop_margin = 0.4 * rect.height;
    this.play_drop_cutoff = rect.top - play_drop_margin;
    this.style.setProperty('--play-drop-margin', `${play_drop_margin.toString()}px`);
    this.classList.remove('hidden');
  }

  setCards(cards: StandardCard[], cards_played: number[] = [], animation_time = 0) {
    let cards_skipped = 0;
    this.cards_container.replaceChildren();
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
      const i_dom = i - cards_skipped;
      el.style.setProperty('--i', i_dom.toString());
      const card_data: CardData = {i, i_dom, el};
      setTimeout(() => {
        // TODO: add card dealt sound effect
        this.cards.set(i, card_data);
        this.cards_container.style.setProperty('--num-cards', this.cards.size.toString());
        this.cards_container.appendChild(el);

        el.addEventListener('mousedown', (e) => {
          if (e.button !== 0) {
            return;
          }
          e.stopImmediatePropagation();
          this.startDraggingCard(e, card_data);
        });

        el.addEventListener('mouseenter', (e) => {
          e.stopImmediatePropagation();
          el.classList.add('hovering');
        });

        el.addEventListener('mouseleave', (e) => {
          e.stopImmediatePropagation();
          el.classList.remove('hovering');
        });

        el.addEventListener('touchstart', (e) => {
          e.stopImmediatePropagation();
          if (e.touches.length === 0) {
            return;
          }
          const touch = e.touches[0];
          this.startDraggingCard({x: touch.clientX, y: touch.clientY}, card_data, touch.identifier);
        });

        el.addEventListener('mouseup', (e) => {
          if (e.button !== 0) {
            return;
          }
          e.stopImmediatePropagation();
          el.classList.remove('hovering');
          if (!this.dragging_data.dragging) {
            return;
          }
          if (this.dragging_data.index !== i) {
            return;
          }
          this.stopDraggingCard(card_data);
        });

        el.addEventListener('touchend', (e) => {
          e.stopImmediatePropagation();
          if (!this.dragging_data.dragging) {
            return;
          }
          if (e.touches.length === 0) {
            this.stopDraggingCard(card_data);
            return;
          }
          const touch = e.touches[0];
          if (touch.identifier !== this.dragging_data.touch_identifier) {
            return;
          }
          this.stopDraggingCard(card_data);
        });

      }, (1 + i) * animation_time);
    }

    document.body.addEventListener('mousemove', (e) => {
      if (!this.dragging_data.dragging) {
        return;
      }
      const card = this.cards.get(this.dragging_data.index);
      if (!card) {
        this.stopDragging();
        return;
      }
      e.stopImmediatePropagation();
      this.dragCard(e, card);
    });

    document.documentElement.addEventListener('mouseenter', (e) => {
      if (!this.dragging_data.dragging || e.buttons % 2 === 1) {
        return;
      }
      const card = this.cards.get(this.dragging_data.index);
      if (!card) {
        this.stopDragging();
        return;
      }
      this.stopDraggingCard(card);
    });

    document.body.addEventListener('touchmove', (e) => {
      e.stopImmediatePropagation();
      if (!this.dragging_data.dragging) {
        return;
      }
      const card = this.cards.get(this.dragging_data.index);
      if (!card) {
        this.stopDragging();
        return;
      }
      if (e.touches.length === 0) {
        return;
      }
      const touch = e.touches[0];
      if (touch.identifier !== this.dragging_data.touch_identifier) {
        return;
      }
      this.dragCard({x: touch.clientX, y: touch.clientY}, card);
    });
  }

  private startDraggingCard(p: Point2D, card: CardData, touch_identifier = 0) {
    this.dragging_lock(async () => {
      if (this.dragging_data.dragging) {
        const previous_card = this.cards.get(this.dragging_data.index);
        if (!!previous_card) {
          this.stopDraggingCard(previous_card);
        }
      }
      this.dragging_data.dragging = true;
      this.dragging_data.index = card.i;
      this.dragging_data.touch_identifier = touch_identifier;
      this.dragging_data.start = p;
      card.el.style.setProperty('--x', '0px');
      card.el.style.setProperty('--y', '0px');
      this.play_drop.classList.remove('hovering');
      this.cancel_drop.classList.add('hovering');
      this.dragging_data.hovering_play_drop = false;
      card.el.classList.add('dragging');
      this.classList.add('dragging');
    });
  }

  private dragCard(p: Point2D, card: CardData) {
    this.dragging_lock(async () => {
      card.el.style.setProperty('--x', `${(p.x - this.dragging_data.start.x).toString()}px`);
      card.el.style.setProperty('--y', `${(p.y - this.dragging_data.start.y).toString()}px`);
      if (p.y < this.play_drop_cutoff) {
        this.play_drop.classList.add('hovering');
        this.cancel_drop.classList.remove('hovering');
        this.dragging_data.hovering_play_drop = true;
      } else {
        this.play_drop.classList.remove('hovering');
        this.cancel_drop.classList.add('hovering');
        this.dragging_data.hovering_play_drop = false;
      }
    });
  }

  private stopDragging() {
    this.dragging_data.dragging = false;
    this.classList.remove('dragging');
  }

  private stopDraggingCard(card: CardData) {
    this.dragging_lock(async () => {
      this.stopDragging();
      card.el.classList.remove('dragging');
      if (this.dragging_data.hovering_play_drop) {
        this.dispatchEvent(new CustomEvent<number>('play_card', {'detail': card.i}));
      }
    });
  }

  playCard(index: number) {
    const card = this.cards.get(index);
    if (!card) {
      console.error('Trying to play card that does not exist');
      return;
    }
    card.el.remove();
    this.cards.delete(index);
    this.cards_container.style.setProperty('--num-cards', this.cards.size.toString());
    for (const other_card of this.cards.values()) {
      if (other_card.i_dom <= card.i_dom) {
        continue;
      }
      other_card.i_dom--;
      other_card.el.style.setProperty('--i', other_card.i_dom.toString());
    }
  }
}

customElements.define('dwg-card-hand', DwgCardHand);

declare global{
  interface HTMLElementTagNameMap {
    'dwg-card-hand': DwgCardHand;
  }
}
