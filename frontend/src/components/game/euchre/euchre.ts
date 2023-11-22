import {DwgElement} from '../../dwg_element';
import {GameComponent, UpdateMessage} from '../data_models';
import {DwgCardHand} from '../util/card_hand/card_hand';
import {createMessage} from '../../lobby/data_models';
import {clientOnMobile, until, untilTimer} from '../../../scripts/util';
import {StandardCard, cardToIcon, cardToImagePath} from '../util/card_util';

import html from './euchre.html';
import {DwgEuchrePlayer} from './euchre_player/euchre_player';
import {BidChooseTrump, DealRound, DealerSubstitutesCard, GameEuchre, PlayCard, PlayerBid, getPlayersTeam} from './euchre_data';

import './euchre.scss';
import './euchre_player/euchre_player';
import '../../dialog_box/message_dialog/message_dialog';
import '../util/card_hand/card_hand';

export class DwgEuchre extends DwgElement implements GameComponent {
  round_number: HTMLSpanElement;
  trick_number: HTMLSpanElement;
  status_container: HTMLSpanElement;
  card_face_up_img: HTMLImageElement;
  table_container: HTMLDivElement;
  trick_cards: HTMLDivElement;
  player_container: HTMLDivElement;
  players_cards: DwgCardHand;

  game: GameEuchre;
  current_trick = 0;
  player_els: DwgEuchrePlayer[] = [];
  player_id: number = -1;
  trick_card_els: HTMLDivElement[] = [];

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('round_number');
    this.configureElement('trick_number');
    this.configureElement('status_container');
    this.configureElement('card_face_up_img');
    this.configureElement('table_container');
    this.configureElement('trick_cards');
    this.configureElement('player_container');
    this.configureElement('players_cards');
  }

  protected override parsedCallback(): void {
    this.round_number.innerText = '-';
    this.trick_number.innerText = '-';
    this.status_container.innerText = 'Starting game ...';
    this.players_cards.addEventListener('play_card', (e: CustomEvent<number>) => {
      // TODO: check if card is playable
      const game_update = createMessage(`player-${this.player_id}`, 'game-update', `{"index":${e.detail}}`, 'play-card');
      this.dispatchEvent(new CustomEvent('game_update', {'detail': game_update, 'bubbles': true}));
    });
    if (clientOnMobile()) {
      this.table_container.style.setProperty('background-image', 'url(/images/card_table400.png)');
    } else {
      this.table_container.style.setProperty('background-image', 'url(/images/card_table800.png)');
    }
  }

  initialize(game: GameEuchre, client_id: number): void {
    this.game = game;
    this.player_els = [];
    for (const [player_id, player] of game.players.entries()) {
      const player_el = document.createElement('dwg-euchre-player');
      player_el.initialize(player, getPlayersTeam(this.game, player_id));
      this.player_container.appendChild(player_el);
      this.player_els.push(player_el);
      if (player.player.client_id === client_id) {
        this.player_id = player_id;
        player_el.setClientPlayer();
      }
    }
    // TODO: if view just don't use player_id
    if (this.player_id > -1) {
      for (let i = 0; i < this.player_els.length; i++) {
        let id = this.player_id + i;
        if (id >= this.player_els.length) {
          id -= this.player_els.length;
        }
        this.game.players[id].order = i;
        this.player_els[id].style.setProperty('--order', i.toString());
        this.player_els[id].style.setProperty('--num-players', this.player_els.length.toString());
      }
    }
    this.trick_cards.style.setProperty('--num-players', this.player_els.length.toString());
    if (game.game_base.game_ended) {
      // TODO: show ended game state
    }
    else if (game.game_base.game_started) {
      // TODO: implement
      this.round_number.innerText = game.round.toString();
      for (const [player_id, player_el] of this.player_els.entries()) {
        player_el.gameStarted(game.bidding, !game.game_base.game_ended && player_id === game.turn, !game.game_base.game_ended && player_id === game.dealer);
        if (this.player_id == player_id && !game.game_base.game_ended) {
          this.players_cards.setCards(game.players[player_id].cards, game.players[player_id].cards_played);
          if (!game.bidding && this.game.turn === this.player_id) {
            this.players_cards.can_play = true;
          }
        }
      }
      if (game.bidding || game.bidding_choosing_trump || game.dealer_substituting_card) {
        this.current_trick = 0;
        this.trick_number.innerText = '-';
        this.status_container.innerText = `${this.game.players[this.game.turn].player.nickname} Bidding`;
        if (game.bidding) {
          this.setCardFaceUpImage(game.card_face_up);
        } else if (game.bidding_choosing_trump) {
          // TODO: back of card img
        } else {
          // TODO: trump img
        }
      } else {
        this.current_trick = game.teams.reduce((a, b) => a + b.tricks, 1);
        this.trick_number.innerText = this.current_trick.toString();
        this.status_container.innerText = `${this.game.players[this.game.turn].player.nickname} Playing`;
        // TODO: trump img
      }
    }
  }

  async gameUpdate(update: UpdateMessage): Promise<void> {
    try {
      switch(update.kind) {
        case 'deal-round':
          const dealRoundData = update.update as DealRound;
          await this.applyDealRound(dealRoundData);
          break;
        case 'bid':
          const bidData = update.update as PlayerBid;
          if (this.game.turn !== bidData.player_id) {
            // TODO: try to sync data
            throw new Error('Player bid out of order');
          }
          await this.applyPlayerBid(bidData);
          break;
        case 'bid-choose-trump':
          const bidChooseTrumpData = update.update as BidChooseTrump;
          if (this.game.turn !== bidChooseTrumpData.player_id) {
            // TODO: try to sync data
            throw new Error('Player bid choose trump out of order');
          }
          await this.applyBidChooseTrump(bidChooseTrumpData);
          break;
        case 'dealer-substitutes-card':
          const dealerSubstitutesCardData = update.update as DealerSubstitutesCard;
          if (this.game.dealer !== dealerSubstitutesCardData.player_id) {
            // TODO: try to sync data
            throw new Error('Only dealer can substitute card');
          }
          await this.applyDealerSubstitutesCard(dealerSubstitutesCardData);
          break;
        case 'play-card':
          const playCardData = update.update as PlayCard;
          if (this.game.turn !== playCardData.player_id) {
            // TODO: try to sync data
            throw new Error('Player played out of order');
          }
          await this.applyPlayCard(playCardData);
          break;
        default:
          console.log(`Unknown game update type ${update.kind}`);
          break;
      }
    } catch(e) {
      console.log(`Error during game update ${JSON.stringify(update)}: ${e}`);
    }
  }

  private async applyDealRound(data: DealRound) {
    const animation_time = Math.min(150 * data.cards.length, 4000) + 250;
    this.game.bidding = true;
    this.game.dealer = data.dealer;
    this.game.round = data.round;
    this.status_container.innerText = `${this.game.players[this.game.dealer].player.nickname} Dealing`;
    this.round_number.innerText = data.round.toString();
    this.game.card_face_up = data.card_face_up;
    this.setCardFaceUpImage(data.card_face_up);
    for (const [player_id, player_el] of this.player_els.entries()) {
      if (data.dealer === player_id) {
        player_el.newRound(true);
      } else {
        player_el.newRound(false);
      }
      if (player_id === this.player_id) {
        const card_animation_time = Math.min(150, animation_time / data.cards.length);
        this.players_cards.setCards(data.cards, [], card_animation_time);
      }
    }
    await untilTimer(animation_time);
    this.game.turn = this.game.dealer + 1;
    if (this.game.turn >= this.game.players.length) {
      this.game.turn -= this.game.players.length;
    }
    this.game.trick_leader = this.game.turn;
    for (const [player_id, player] of this.game.players.entries()) {
      if (player_id === this.player_id) {
        player.cards = data.cards;
      } else {
        player.cards = [];
      }
    }
    for (const team of this.game.teams) {
      team.tricks = 0;
    }
    this.trick_number.innerText = '-';
    this.current_trick = 0;
    this.player_els[this.game.turn].bidding();
    this.status_container.innerText = `${this.game.players[this.game.turn].player.nickname} Bidding`;
  }

  private async applyPlayerBid(data: PlayerBid) {
    // TODO: await this.player_els[data.player_id].setBidAnimation(data.amount);
    this.player_els[data.player_id].setBid();
    this.game.bidding = false;
    this.game.makers_team = data.player_id % 2;
    this.game.defenders_team = this.game.makers_team == 0 ? 1 : 0;
    this.game.player_bid = data.player_id;
    this.game.going_alone = data.going_alone;
    this.game.turn = this.game.dealer + 1;
    this.resolveTurn();
    this.game.trick_leader = this.game.turn;
    this.game.trump_suit = this.game.card_face_up.suit;
    this.game.dealer_substituting_card = true;
    this.player_els[this.game.dealer].substitutingCard();
    this.status_container.innerText = `${this.game.players[this.game.dealer].player.nickname} Substituting Card`;
  }

  private async applyBidChooseTrump(data: BidChooseTrump) {
    // TODO: await this.player_els[data.player_id].setBidAnimation(data.amount);
    this.player_els[data.player_id].setBid();
    this.game.bidding_choosing_trump = false;
    this.game.makers_team = data.player_id % 2;
    this.game.defenders_team = this.game.makers_team == 0 ? 1 : 0;
    this.game.player_bid = data.player_id;
    this.game.going_alone = data.going_alone;
    this.game.turn = this.game.dealer + 1;
    this.resolveTurn();
    this.game.trick_leader = this.game.turn;
    this.game.trump_suit = data.trump_suit;
    // TODO: set trump img
    this.player_els[this.game.turn].playing();
    this.status_container.innerText = `${this.game.players[this.game.turn].player.nickname} Playing`;
  }

  private async applyDealerSubstitutesCard(data: DealerSubstitutesCard) {
    // TODO: animation
    this.game.dealer_substituting_card = false;
    this.player_els[data.player_id].substitutedCard();
    // TODO: set trump img
    this.player_els[this.game.turn].playing();
    this.status_container.innerText = `${this.game.players[this.game.turn].player.nickname} Playing`;
  }

  private async applyPlayCard(data: PlayCard) {
    if (this.player_id === data.player_id) {
      this.game.players[data.player_id].cards.splice(data.index, 1);
    }
    this.game.trick.push(data.card);
    this.player_els[data.player_id].playCard();
    if (data.player_id === this.player_id) {
      this.players_cards.playCard(data.index);
      this.players_cards.can_play = false;
    }
    // TODO: handle viewers properly
    const card_el = document.createElement('div');
    const card_el_img = document.createElement('img');
    card_el_img.src = cardToImagePath(data.card);
    card_el_img.draggable = false;
    card_el_img.alt = cardToIcon(data.card);
    card_el.appendChild(card_el_img);
    card_el.classList.add('card');
    card_el.style.setProperty('--i', this.game.players[data.player_id].order.toString());
    this.trick_cards.appendChild(card_el);
    this.trick_card_els.push(card_el);
    card_el.style.transitionDuration = '1s';
    await until(() => {
      return !!getComputedStyle(card_el).left;
    }, 20);
    card_el.classList.add('played');
    await untilTimer(1000);
    this.status_container.innerText = '';
    this.game.turn++;
    this.resolveTurn();
    if (this.game.turn !== this.game.trick_leader) {
      this.player_els[this.game.turn].playing();
      this.status_container.innerText = `${this.game.players[this.game.turn].player.nickname} Playing`;
      if (this.game.turn === this.player_id) {
        this.players_cards.can_play = true;
      }
      return;
    }
    // end of trick
    // TODO: implement
  }

  private setCardFaceUpImage(card: StandardCard) {
    this.card_face_up_img.src = cardToImagePath(card);
    this.card_face_up_img.classList.add('show');
  }

  private resolveTurn() {
    if (this.game.turn >= this.game.players.length) {
      this.game.turn -= this.game.players.length;
    }
    if (this.game.going_alone && (Math.round(Math.abs(this.game.turn - this.game.player_bid)) === 2)) {
      this.game.turn++;
      if (this.game.turn >= this.game.players.length) {
        this.game.turn -= this.game.players.length;
      }
    }
  }
}

customElements.define('dwg-euchre', DwgEuchre);

declare global{
  interface HTMLElementTagNameMap {
    'dwg-euchre': DwgEuchre;
  }
}
