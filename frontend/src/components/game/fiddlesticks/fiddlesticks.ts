import {DwgElement} from '../../dwg_element';
import {GameBase, GameComponent, GamePlayer} from '../data_models';
import {StandardCard, cardToImagePath, cardToName} from '../util/card_util';
import {DwgFiddlesticksPlayer} from './fiddlesticks_player/fiddlesticks_player';
import {ServerMessage} from '../../lobby/data_models';

import html from './fiddlesticks.html';

import './fiddlesticks.scss';
import './fiddlesticks_player/fiddlesticks_player';

/** Data describing a game of fiddlesticks */
export declare interface GameFiddlesticks {
  game_base: GameBase;
  players: FiddlesticksPlayer[];
  round: number;
  max_round: number;
  rounds_increasing: boolean;
  dealer: number;
  turn: number;
  betting: boolean;
  trump: StandardCard;
  trick_leader: number;
  trick: StandardCard[];
}

/** Data describing a fiddlesticks player */
export declare interface FiddlesticksPlayer {
  player: GamePlayer;
  cards: StandardCard[];
  score: number;
  bet: number;
  tricks: number;
}

/** Data describing a deal-round game-update */
declare interface DealRound {
  round: number;
  dealer: number;
  trump: StandardCard;
  cards: StandardCard[];
}

/** Data describing a bet game-update */
declare interface PlayerBet {
  amount: number;
  player_id: number;
}

/** Data describing a bet game-update */
declare interface PlayCard {
  index: number;
  card: StandardCard;
  player_id: number;
}

/** Data describing a bet game-update-failed */
declare interface PlayCardFailed {
  player_id: number;
  message: string;
}

export class DwgFiddlesticks extends DwgElement implements GameComponent {
  round_number: HTMLSpanElement;
  trick_number: HTMLSpanElement;
  current_trick = 0;
  status_container: HTMLSpanElement;
  trump_card_img: HTMLImageElement;
  trick_cards: HTMLDivElement;
  player_container: HTMLDivElement;

  game: GameFiddlesticks;
  player_els: DwgFiddlesticksPlayer[] = [];
  player_id: number = -1;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('round_number');
    this.configureElement('trick_number');
    this.configureElement('status_container');
    this.configureElement('trump_card_img');
    this.configureElement('trick_cards');
    this.configureElement('player_container');
  }

  protected override parsedCallback(): void {
    this.round_number.innerText = '-';
    this.trick_number.innerText = '-';
    this.status_container.innerText = 'Starting game ...';
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
  }

  gameUpdate(update: ServerMessage): void {
    try {
      switch(update.data) {
        case "deal-round":
          const dealRoundData = JSON.parse(update.content) as DealRound;
          this.round_number.innerText = dealRoundData.round.toString();
          this.trump_card_img.src = cardToImagePath(dealRoundData.trump);
          this.game.trump = dealRoundData.trump;
          for (const [player_id, player_el] of this.player_els.entries()) {
            if (dealRoundData.dealer === player_id) {
              player_el.setDealer(true);
            } else {
              player_el.setDealer(false);
            }
            if (player_id === this.player_id) {
              player_el.setCards(dealRoundData.cards);
            } else {
              player_el.setHiddenCards(dealRoundData.round);
            }
          }
          this.game.betting = true;
          this.game.dealer = dealRoundData.dealer;
          this.game.round = dealRoundData.round;
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
              player.cards = dealRoundData.cards;
            } else {
              player.cards = [];
            }
            player.tricks = 0;
          }
          if (this.game.turn === this.player_id) {
            this.player_els[this.player_id].betting();
          }
          this.status_container.innerText = `${this.game.players[this.game.turn].player.nickname} Betting`;
          break;
        case "bet":
          const betData = JSON.parse(update.content) as PlayerBet;
          if (this.game.turn !== betData.player_id) {
            // TODO: try to sync data
            throw new Error('Player bet out of order');
          }
          this.game.players[betData.player_id].bet = betData.amount;
          this.player_els[betData.player_id].setBet(betData.amount);
          let still_betting = this.game.turn !== this.game.dealer;
          this.game.turn++;
          if (this.game.turn >= this.game.players.length) {
            this.game.turn -= this.game.players.length;
          }
          if (still_betting) {
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
              }
            }
          }
          break;
        case "play-card":
          const playCardData = JSON.parse(update.content) as PlayCard;
          if (this.game.turn !== playCardData.player_id) {
            // TODO: try to sync data
            throw new Error('Player played out of order');
          }
          if (this.player_id === playCardData.player_id) {
            this.game.players[playCardData.player_id].cards.splice(playCardData.index, 1);
          }
          this.game.trick.push(playCardData.card);
          this.player_els[playCardData.player_id].playCard(playCardData.index, playCardData.card);
          this.game.turn++;
          if (this.game.turn >= this.game.players.length) {
            this.game.turn -= this.game.players.length;
          }
          if (this.game.turn === this.game.trick_leader) {
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
            this.game.turn = this.game.trick_leader + winning_index;
            if (this.game.turn >= this.game.players.length) {
              this.game.turn -= this.game.players.length;
            }
            this.game.trick_leader = this.game.turn;
            this.game.players[this.game.turn].tricks++;
            for (const [i, player_el] of this.player_els.entries()) {
              player_el.endTrick(this.game.players[i].tricks);
            }
            this.game.trick = [];
            console.log(`Trick won by ${this.game.players[this.game.turn].player.nickname} with the ${cardToName(winning_card)}`);
            if (this.current_trick === this.game.round) {
              if (this.game.rounds_increasing && this.game.round === this.game.max_round) {
                this.game.round--;
                this.game.rounds_increasing = false;
              } else if (this.game.round === 1) {
                // TODO: game over logic
              } else {
                this.game.round++; // wait for deal-round update from server
              }
              for (const [i, player] of this.game.players.entries()) {
                if (player.bet === player.tricks) {
                  player.score += 10 + player.bet;
                  this.player_els[i].setScore(player.score);
                  this.player_els[i].newRound();
                }
              }
            } else {
              this.current_trick++;
              this.trick_number.innerText = this.current_trick.toString();
              this.player_els[this.game.turn].playing();
              this.status_container.innerText = `${this.game.players[this.game.turn].player.nickname} Playing`;
            }
          } else {
            this.player_els[this.game.turn].playing();
            this.status_container.innerText = `${this.game.players[this.game.turn].player.nickname} Playing`;
          }
          break;
        case "play-card-failed":
          const playCardFailedData = JSON.parse(update.content) as PlayCardFailed;
          if (this.player_id === playCardFailedData.player_id) {
            this.player_els[playCardFailedData.player_id].playing();
            // TODO: show message to user
          }
          break;
        default:
          console.log(`Unknown game update type ${update.data} from ${update.sender}`);
          break;
      }
    } catch(e) {
      console.log(`Error during game update ${JSON.stringify(update)}: ${e}`);
    }
  }
}

customElements.define('dwg-fiddlesticks', DwgFiddlesticks);
