import { DwgElement } from '../../../../dwg_element';
import { createMessage } from '../../../../lobby/data_models';
import type { EgyptianRatCrapPlayer } from '../egyptian_rat_crap_data';
import { Sounds } from '../../../../../sounds/Sounds';

import html from './egyptian_rat_crap_player.html';

import './egyptian_rat_crap_player.scss';

export class DwgEgyptianRatCrapPlayer extends DwgElement {
  private name_container!: HTMLDivElement;
  private status_container!: HTMLDivElement;
  private pile_count_container!: HTMLSpanElement;
  private winner_wrapper!: HTMLDivElement;
  private action_wrapper!: HTMLDivElement;
  private flip_button!: HTMLButtonElement;

  private initialized = false;
  private player!: EgyptianRatCrapPlayer;
  private client_player = false;

  constructor() {
    super();
    this.html_string = html;
    this.configureElements(
      'name_container',
      'status_container',
      'pile_count_container',
      'winner_wrapper',
      'action_wrapper',
      'flip_button'
    );
  }

  protected override parsedCallback(): void {
    if (!this.initialized) {
      throw new Error('Should initialize egyptian rat crap player before attaching to dom');
    }
    this.name_container.innerText = this.player.player.nickname;
    this.pile_count_container.innerText = this.player.pile_count.toString();
  }

  initialize(player: EgyptianRatCrapPlayer): void {
    this.player = player;
    this.initialized = true;
  }

  setClientPlayer(): void {
    this.classList.add('client-player');
    this.client_player = true;
    this.action_wrapper.classList.add('show');
    this.flip_button.addEventListener('click', (e) => {
      e.stopPropagation();
      this.sendFlipEvent();
    });
  }

  sendFlipEvent(): void {
    if (this.flip_button.disabled) {
      return;
    }
    const game_update = createMessage(`player-${this.player.player.player_id}`, 'game-update', '{}', 'flip-card');
    this.dispatchEvent(new CustomEvent('game_update', { detail: game_update, bubbles: true }));
  }

  setTurn(active: boolean): void {
    this.classList.toggle('turn', active);
    if (this.client_player) {
      this.flip_button.disabled = !active;
      if (active) {
        Sounds.play('turn_notification');
      }
    }
  }

  setPileCount(count: number): void {
    this.player.pile_count = count;
    this.pile_count_container.innerText = count.toString();
  }

  setSlapped(slapped: boolean): void {
    this.player.slapped = slapped;
    this.classList.toggle('slapped', slapped);
  }

  wonGame(): void {
    this.winner_wrapper.classList.add('show');
  }
}

customElements.define('dwg-egyptian-rat-crap-player', DwgEgyptianRatCrapPlayer);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-egyptian-rat-crap-player': DwgEgyptianRatCrapPlayer;
  }
}
