import { DwgElement } from '../../../dwg_element';
import type { GameComponent, UpdateMessage } from '../../data_models';
import type { DwgGame } from '../../game';
import { createMessage } from '../../../lobby/data_models';
import { modulus } from '../../../../scripts/math';
import { cardToIcon, cardToImagePath } from '../../util/card_util';
import type { StandardCard } from '../../util/card_util';

import html from './egyptian_rat_crap.html';
import type {
  GameEgyptianRatCrap,
  Deal,
  FlipCard,
  PileAwarded,
  TurnUpdate,
  SlapResult,
} from './egyptian_rat_crap_data';

import './egyptian_rat_crap.scss';
import './egyptian_rat_crap_player/egyptian_rat_crap_player';
import type { DwgEgyptianRatCrapPlayer } from './egyptian_rat_crap_player/egyptian_rat_crap_player';

export class DwgEgyptianRatCrap extends DwgElement implements GameComponent {
  private chances_number!: HTMLSpanElement;
  private status_container!: HTMLSpanElement;
  private central_pile!: HTMLDivElement;
  private player_container!: HTMLDivElement;

  private game!: GameEgyptianRatCrap;
  private player_els: DwgEgyptianRatCrapPlayer[] = [];
  private player_id: number = -1;

  constructor() {
    super();
    this.html_string = html;
    this.configureElements('chances_number', 'status_container', 'central_pile', 'player_container');
  }

  protected override parsedCallback(): void {
    this.chances_number.innerText = '-';
    this.status_container.innerText = 'Starting game ...';
    document.addEventListener('keydown', (e) => {
      if (e.code === 'Space' || e.key === ' ') {
        e.preventDefault();
        this.sendSlapEvent();
      }
    });
    this.addEventListener('click', () => {
      this.sendSlapEvent();
    });
  }

  async initialize(abstract_game: DwgGame, game: GameEgyptianRatCrap): Promise<void> {
    this.player_id = abstract_game.playerId();
    this.game = game;
    this.player_els = [];
    for (const [player_id, player] of game.players.entries()) {
      const player_el = document.createElement('dwg-egyptian-rat-crap-player');
      player_el.initialize(player);
      this.player_container.appendChild(player_el);
      this.player_els.push(player_el);
      if (this.player_id === player_id) {
        player_el.setClientPlayer();
      }
    }
    for (let i = 0; i < this.player_els.length; i++) {
      const id = modulus(this.player_id + i, this.game.players.length);
      const order = abstract_game.isPlayer() ? i : i + 0.5;
      this.game.players[id].order = order;
      this.player_els[id].style.setProperty('--order', order.toString());
      this.player_els[id].style.setProperty('--num-players', this.player_els.length.toString());
    }
    if (game.game_base.game_started && !game.game_base.game_ended && game.turn >= 0) {
      this.chances_number.innerText = game.challenge_active ? game.chances_left.toString() : '-';
      this.setCentralPileImage();
      for (const [player_id, player_el] of this.player_els.entries()) {
        player_el.setPileCount(this.game.players[player_id].pile_count);
        player_el.setTurn(!game.pile_pending && player_id === game.turn);
      }
      this.status_container.innerText = game.pile_pending
        ? 'Resolving ...'
        : `${this.game.players[game.turn].player.nickname} Playing`;
    }
  }

  private setCentralPileImage() {
    const top = this.game.central_pile[this.game.central_pile.length - 1];
    this.central_pile.classList.toggle('slappable', isValidSlap(this.game.central_pile));
    if (!top) {
      this.central_pile.replaceChildren();
      return;
    }
    const img = document.createElement('img');
    img.src = cardToImagePath(top);
    img.draggable = false;
    img.alt = cardToIcon(top, false);
    this.central_pile.replaceChildren(img);
  }

  private sendSlapEvent() {
    if (this.player_id < 0 || this.game.players[this.player_id]?.slapped) {
      return;
    }
    const game_update = createMessage(`player-${this.player_id}`, 'game-update', '{}', 'slap');
    this.dispatchEvent(new CustomEvent('game_update', { detail: game_update, bubbles: true }));
  }

  async gameUpdate(update: UpdateMessage): Promise<void> {
    try {
      switch (update.kind) {
        case 'deal':
          this.applyDeal(update.content as Deal);
          break;
        case 'flip-card':
          this.applyFlipCard(update.content as FlipCard);
          break;
        case 'pile-awarded':
          this.applyPileAwarded(update.content as PileAwarded);
          break;
        case 'turn-update':
          this.applyTurnUpdate(update.content as TurnUpdate);
          break;
        case 'slap-result':
          this.applySlapResult(update.content as SlapResult);
          break;
        default:
          console.log(`Unknown game update type ${update.kind}`);
          break;
      }
    } catch (e) {
      console.log(`Error during game update ${JSON.stringify(update)}: ${e}`);
    }
  }

  private applyDeal(data: Deal) {
    this.game.central_pile = [];
    this.game.challenge_active = false;
    this.game.chances_left = 0;
    this.setCentralPileImage();
    this.chances_number.innerText = '-';
    for (const [player_id, player_el] of this.player_els.entries()) {
      player_el.setPileCount(data.pile_counts[player_id]);
      player_el.setSlapped(false);
      player_el.setTurn(player_id === data.turn);
    }
    this.game.turn = data.turn;
    this.status_container.innerText = `${this.game.players[data.turn].player.nickname} Playing`;
  }

  private applyFlipCard(data: FlipCard) {
    this.game.players[data.player_id].pile_count--;
    this.player_els[data.player_id].setPileCount(this.game.players[data.player_id].pile_count);
    this.game.central_pile.push(data.card);
    this.setCentralPileImage();
    for (const player_el of this.player_els) {
      player_el.setSlapped(false);
    }
    if (data.challenger_id !== undefined) {
      this.game.challenge_active = true;
      this.game.challenger_id = data.challenger_id;
      this.game.chances_left = data.chances_left ?? 0;
      this.chances_number.innerText = this.game.chances_left.toString();
      this.status_container.innerText = `${this.game.players[data.challenger_id].player.nickname} played a challenge card!`;
    } else if (data.challenge_busted) {
      this.game.pile_pending = true;
      this.status_container.innerText = 'Challenge busted, resolving ...';
    } else if (data.chances_left !== undefined) {
      this.game.chances_left = data.chances_left;
      this.chances_number.innerText = data.chances_left.toString();
    } else {
      this.game.challenge_active = false;
      this.chances_number.innerText = '-';
    }
    for (const [player_id, player_el] of this.player_els.entries()) {
      player_el.setTurn(!data.challenge_busted && player_id === data.turn);
    }
    this.game.turn = data.turn;
    if (!data.challenge_busted && data.turn >= 0) {
      this.status_container.innerText = `${this.game.players[data.turn].player.nickname} Playing`;
    }
  }

  private applyPileAwarded(data: PileAwarded) {
    this.game.players[data.pile_awarded_to].pile_count += data.pile_size;
    this.player_els[data.pile_awarded_to].setPileCount(this.game.players[data.pile_awarded_to].pile_count);
    this.game.central_pile = [];
    this.game.challenge_active = false;
    this.setCentralPileImage();
    this.chances_number.innerText = '-';
    this.status_container.innerText = `${this.game.players[data.pile_awarded_to].player.nickname} won ${data.pile_size} cards!`;
    for (const player_el of this.player_els) {
      player_el.setTurn(false);
    }
  }

  private applyTurnUpdate(data: TurnUpdate) {
    this.game.turn = data.turn;
    this.game.pile_pending = false;
    for (const [player_id, player_el] of this.player_els.entries()) {
      player_el.setTurn(player_id === data.turn);
    }
    this.status_container.innerText = `${this.game.players[data.turn].player.nickname} Playing`;
  }

  private applySlapResult(data: SlapResult) {
    if (data.valid && data.pile_size !== undefined) {
      this.game.players[data.player_id].pile_count += data.pile_size;
      this.player_els[data.player_id].setPileCount(this.game.players[data.player_id].pile_count);
      this.game.central_pile = [];
      this.game.challenge_active = false;
      this.setCentralPileImage();
      this.chances_number.innerText = '-';
      this.status_container.innerText = `${this.game.players[data.player_id].player.nickname} slapped and won ${data.pile_size} cards!`;
      for (const player_el of this.player_els) {
        player_el.setTurn(false);
      }
    } else if (data.pile_empty) {
      this.status_container.innerText = `${this.game.players[data.player_id].player.nickname} missed the slap`;
    } else {
      this.game.players[data.player_id].pile_count--;
      this.player_els[data.player_id].setPileCount(this.game.players[data.player_id].pile_count);
      this.status_container.innerText = `${this.game.players[data.player_id].player.nickname} slapped invalidly!`;
    }
    this.player_els[data.player_id].setSlapped(true);
  }

  updateDialogComponent(update: UpdateMessage): HTMLElement {
    const update_el = document.createElement('div');
    update_el.innerText = `ID: ${update.update_id}, Kind: ${update.kind}, data: ${JSON.stringify(update.content)}`;
    return update_el;
  }
}

// mirrors the backend's isValidSlap in egyptian_rat_crap.go
function isValidSlap(central_pile: StandardCard[]): boolean {
  const n = central_pile.length;
  if (n < 2) {
    return false;
  }
  const top = central_pile[n - 1];
  const second = central_pile[n - 2];
  if (top.number === second.number) {
    return true; // doubles
  }
  if (n >= 3 && top.number === central_pile[n - 3].number) {
    return true; // sandwich
  }
  if (top.number + second.number === 10) {
    return true; // sum to 10
  }
  return (top.number === 12 && second.number === 13) || (top.number === 13 && second.number === 12); // marriage
}

customElements.define('dwg-egyptian-rat-crap', DwgEgyptianRatCrap);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-egyptian-rat-crap': DwgEgyptianRatCrap;
  }
}
