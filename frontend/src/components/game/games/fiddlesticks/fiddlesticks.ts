import {DwgElement} from '../../../dwg_element';
import {GameComponent, UpdateMessage} from '../../data_models';
import {StandardCard, cardToIcon, cardToImagePath, cardToName} from '../../util/card_util';
import {DwgFiddlesticksPlayer} from './fiddlesticks_player/fiddlesticks_player';
import {DwgCardHand} from '../../util/card_hand/card_hand';
import {createMessage} from '../../../lobby/data_models';
import {clientOnMobile, until, untilTimer} from '../../../../scripts/util';
import {messageDialog} from '../../game';

import html from './fiddlesticks.html';
import {GameFiddlesticks, DealRound, PlayerBet, PlayCard} from './fiddlesticks_data';

import './fiddlesticks.scss';
import './fiddlesticks_player/fiddlesticks_player';
import '../../../dialog_box/message_dialog/message_dialog';
import '../../util/card_hand/card_hand';

export class DwgFiddlesticks extends DwgElement implements GameComponent {
  round_number: HTMLSpanElement;
  bets_number: HTMLSpanElement;
  trick_number: HTMLSpanElement;
  status_container: HTMLSpanElement;
  trump_card_img: HTMLImageElement;
  trick_cards: HTMLDivElement;
  table_container: HTMLDivElement;
  player_container: HTMLDivElement;
  players_cards: DwgCardHand;

  game: GameFiddlesticks;
  current_trick = 0;
  player_els: DwgFiddlesticksPlayer[] = [];
  player_id: number = -1;
  trick_card_els: HTMLDivElement[] = [];

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('round_number');
    this.configureElement('bets_number');
    this.configureElement('trick_number');
    this.configureElement('status_container');
    this.configureElement('trump_card_img');
    this.configureElement('trick_cards');
    this.configureElement('table_container');
    this.configureElement('player_container');
    this.configureElement('players_cards');
  }

  protected override parsedCallback(): void {
    this.round_number.innerText = '-';
    this.bets_number.innerText = '-';
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

  initialize(game: GameFiddlesticks, client_id: number): void {
    this.game = game;
    this.player_els = [];
    for (const [player_id, player] of game.players.entries()) {
      const player_el = document.createElement('dwg-fiddlesticks-player');
      player_el.initialize(player);
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
      this.round_number.innerText = game.round.toString();
      this.updateBetsContainer();
      for (const [player_id, player_el] of this.player_els.entries()) {
        player_el.gameStarted(game.betting, !game.game_base.game_ended && player_id === game.turn, !game.game_base.game_ended && player_id === game.dealer);
        if (this.player_id == player_id && !game.game_base.game_ended) {
          this.players_cards.setCards(game.players[player_id].cards, game.players[player_id].cards_played);
          if (!game.betting && this.game.turn === this.player_id) {
            this.players_cards.can_play = true;
          }
        }
      }
      if (game.betting) {
        this.current_trick = 0;
        this.trick_number.innerText = '-';
        this.status_container.innerText = `${this.game.players[this.game.turn].player.nickname} Betting`;
      } else {
        this.current_trick = game.players.reduce((a, b) => a + b.tricks, 1);
        this.trick_number.innerText = this.current_trick.toString();
        for (const [i, card] of game.trick.entries()) {
          let player_id = game.trick_leader + i;
          if (player_id >= game.players.length) {
            player_id -= game.players.length;
          }
          this.addPlayedCard({card, index: i, player_id});
        }
        this.status_container.innerText = `${this.game.players[this.game.turn].player.nickname} Playing`;
      }
      this.setTrumpImage(game.trump);
    }
  }

  private updateBetsContainer() {
    let bets = 0;
    let t = this.game.dealer + 1;
    while (true) {
      if (t >= this.game.players.length) {
        t -= this.game.players.length;
      }
      if (this.game.betting && t === this.game.turn) {
        break;
      }
      bets += this.game.players[t].bet;
      if (t === this.game.dealer) {
        break;
      }
      t++;
    }
    this.bets_number.innerText = bets.toString();
  }

  setTrumpImage(trump: StandardCard) {
    this.trump_card_img.src = cardToImagePath(trump);
    this.trump_card_img.classList.add('show');
  }

  async gameUpdate(update: UpdateMessage): Promise<void> {
    try {
      switch(update.kind) {
        case "deal-round":
          const dealRoundData = update.update as DealRound;
          await this.applyDealRound(dealRoundData);
          break;
        case "bet":
          const betData = update.update as PlayerBet;
          if (this.game.turn !== betData.player_id) {
            // TODO: try to sync data
            throw new Error('Player bet out of order');
          }
          await this.applyBet(betData);
          break;
        case "play-card":
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
    this.game.betting = true;
    this.game.dealer = data.dealer;
    this.game.round = data.round;
    this.status_container.innerText = `${this.game.players[this.game.dealer].player.nickname} Dealing`;
    this.round_number.innerText = data.round.toString();
    this.bets_number.innerText = '0';
    this.setTrumpImage(data.trump);
    this.game.trump = data.trump;
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
    this.trick_number.innerText = '-';
    this.current_trick = 0;
    if (this.game.round === this.game.max_round) {
      this.game.rounds_increasing = false;
    }
    this.game.turn = this.game.dealer + 1;
    if (this.game.turn >= this.game.players.length) {
      this.game.turn -= this.game.players.length;
    }
    this.game.trick_leader = this.game.turn;
    for (const [player_id, player] of this.game.players.entries()) {
      player.bet = -1;
      if (player_id === this.player_id) {
        player.cards = data.cards;
      } else {
        player.cards = [];
      }
      player.tricks = 0;
    }
    this.player_els[this.game.turn].betting();
    this.status_container.innerText = `${this.game.players[this.game.turn].player.nickname} Betting`;
  }

  private async applyBet(data: PlayerBet) {
    this.game.players[data.player_id].bet = data.amount;
    await this.player_els[data.player_id].setBetAnimation(data.amount);
    this.game.betting = this.game.turn !== this.game.dealer;
    this.status_container.innerText = '';
    this.game.turn++;
    if (this.game.turn >= this.game.players.length) {
      this.game.turn -= this.game.players.length;
    }
    this.updateBetsContainer();
    if (this.game.betting) {
      this.player_els[this.game.turn].betting();
      this.status_container.innerText = `${this.game.players[this.game.turn].player.nickname} Betting`;
    } else {
      this.game.trick_leader = this.game.turn;
      this.trick_number.innerText = '1';
      this.current_trick = 1;
      this.status_container.innerText = `${this.game.players[this.game.turn].player.nickname} Playing`;
      for (const [i, player_el] of this.player_els.entries()) {
        player_el.endBetting();
        if (i === this.game.turn) {
          player_el.playing();
          if (i === this.player_id) {
            this.players_cards.can_play = true;
          }
        }
      }
    }
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
    await this.addPlayedCard(data);
    await untilTimer(1000);
    this.status_container.innerText = '';
    this.game.turn++;
    if (this.game.turn >= this.game.players.length) {
      this.game.turn -= this.game.players.length;
    }
    if (this.game.turn !== this.game.trick_leader) {
      this.player_els[this.game.turn].playing();
      this.status_container.innerText = `${this.game.players[this.game.turn].player.nickname} Playing`;
      if (this.game.turn === this.player_id) {
        this.players_cards.can_play = true;
      }
      return;
    }
    // end of trick
    await untilTimer(500);
    let winning_index = 0;
    let winning_card = this.game.trick[0];
    for (let i = 1; i < this.game.trick.length; i++) {
      const card = this.game.trick[i];
      if (card.suit === winning_card.suit) {
        if (card.number > winning_card.number) {
          winning_index = i;
          winning_card = card;
        }
      } else if (card.suit === this.game.trump.suit) {
        winning_index = i;
        winning_card = card;
      }
    }
    if (winning_index >= 0 && winning_index < this.trick_card_els.length) {
      this.trick_card_els[winning_index].style.zIndex = '2';
    }
    this.game.turn = this.game.trick_leader + winning_index;
    if (this.game.turn >= this.game.players.length) {
      this.game.turn -= this.game.players.length;
    }
    this.game.trick_leader = this.game.turn;
    for (const trick_card_el of this.trick_card_els) {
      trick_card_el.classList.remove('played');
      trick_card_el.classList.add('center');
    }
    await untilTimer(1000);
    this.trick_cards.style.setProperty('--winner', this.game.players[this.game.turn].order.toString());
    for (const trick_card_el of this.trick_card_els) {
      trick_card_el.classList.remove('center');
      trick_card_el.classList.add('winner');
    }
    await untilTimer(1000);
    this.game.players[this.game.turn].tricks++;
    for (const [i, player_el] of this.player_els.entries()) {
      player_el.endTrick(this.game.players[i].tricks);
    }
    this.trick_cards.replaceChildren();
    this.trick_card_els = [];
    this.game.trick = [];
    this.trick_number.innerText = '-';
    console.log(`Trick won by ${this.game.players[this.game.turn].player.nickname} with the ${cardToName(winning_card)}`);
    if (this.current_trick !== this.game.round) {
      this.current_trick++;
      this.trick_number.innerText = this.current_trick.toString();
      this.player_els[this.game.turn].playing();
      this.status_container.innerText = `${this.game.players[this.game.turn].player.nickname} Playing`;
      if (this.game.turn === this.player_id) {
        this.players_cards.can_play = true;
      }
      return;
    }
    // end of round
    await untilTimer(500);
    // TODO: score animations for a few seconds
    this.bets_number.innerText = '-';
    this.trump_card_img.classList.remove('show');
    for (const [i, player] of this.game.players.entries()) {
      if (player.bet === player.tricks) {
        player.score += this.game.round_points + this.game.trick_points * player.bet;
        this.player_els[i].setScore(player.score);
      }
      this.player_els[i].endRound();
    }
    this.round_number.innerText = '-';
    if (this.game.rounds_increasing && this.game.round === this.game.max_round) {
      this.game.rounds_increasing = false;
    }
    if (!this.game.rounds_increasing && this.game.round === 1) {
      // end of game
      this.game.game_base.game_ended = true;
      let winners = [0];
      let winning_score = this.game.players[0].score;
      for (let i = 1; i < this.game.players.length; i++) {
        const player = this.game.players[i];
        if (player.score > winning_score) {
          winners = [i];
          winning_score = player.score;
        } else if (player.score === winning_score) {
          winners.push(i);
        }
      }
      for (const winner of winners) {
        this.player_els[winner].wonGame();
      }
      let winner_text = winners.length > 1 ? 'The winners are: ' : 'The winner is: ';
      winner_text += winners.map(winner => this.game.players[winner].player.nickname).join(', ');
      winner_text += `\nWith ${winning_score} points`;
      messageDialog.call(this, {message: winner_text});
      this.status_container.innerText = 'game over';
    } else if (this.game.rounds_increasing) {
      this.game.round++; // wait for deal-round update from server
    } else {
      this.game.round--; // wait for deal-round update from server
    }
  }

  private async addPlayedCard(data: PlayCard) {
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
  }
}

customElements.define('dwg-fiddlesticks', DwgFiddlesticks);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-fiddlesticks': DwgFiddlesticks;
  }
}
