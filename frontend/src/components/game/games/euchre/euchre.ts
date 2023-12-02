import {DwgElement} from '../../../dwg_element';
import {GameComponent, UpdateMessage} from '../../data_models';
import {DwgCardHand} from '../../util/card_hand/card_hand';
import {createMessage} from '../../../lobby/data_models';
import {clientOnMobile, until, untilTimer} from '../../../../scripts/util';
import {modulus} from '../../../../scripts/math';
import {StandardCard, cardSuitToColor, cardSuitToName, cardToIcon, cardToImagePath, cardToName} from '../../util/card_util';
import {DwgGame, messageDialog} from '../../game';

import html from './euchre.html';
import {DwgEuchrePlayer} from './euchre_player/euchre_player';
import {BidChooseTrump, DealRound, DealerSubstitutesCard, GameEuchre, PlayCard, PlayerBid, PlayerPass, getPlayersTeam} from './euchre_data';

import './euchre.scss';
import './euchre_player/euchre_player';
import '../../../dialog_box/message_dialog/message_dialog';
import '../../util/card_hand/card_hand';

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
      const game_update = createMessage(`player-${this.player_id}`, 'game-update',
        `{"index":${e.detail}}`, this.game.dealer_substituting_card ? 'dealer-substitutes-card' : 'play-card');
      this.dispatchEvent(new CustomEvent('game_update', {'detail': game_update, 'bubbles': true}));
    });
    if (clientOnMobile()) {
      this.table_container.style.setProperty('background-image', 'url(/images/card_table400.png)');
    } else {
      this.table_container.style.setProperty('background-image', 'url(/images/card_table800.png)');
    }
  }

  initialize(abstract_game: DwgGame, game: GameEuchre): void {
    this.player_id = abstract_game.player_id;
    this.game = game;
    this.player_els = [];
    for (const [player_id, player] of game.players.entries()) {
      const player_el = document.createElement('dwg-euchre-player');
      player_el.initialize(player, getPlayersTeam(this.game, player_id));
      this.player_container.appendChild(player_el);
      this.player_els.push(player_el);
      if (this.player_id === player_id) {
        player_el.setClientPlayer();
      }
    }
    for (let i = 0; i < this.player_els.length; i++) {
      const id = modulus(this.player_id + i, this.game.players.length);
      const order = abstract_game.is_player ? i : i + 0.5;
      this.game.players[id].order = order;
      this.player_els[id].style.setProperty('--order', order.toString());
      this.player_els[id].style.setProperty('--num-players', this.player_els.length.toString());
    }
    this.trick_cards.style.setProperty('--num-players', this.player_els.length.toString());
    if (game.game_base.game_ended) {
      // TODO: show ended game state
    }
    else if (game.game_base.game_started) {
      this.round_number.innerText = game.round.toString();
      for (const [player_id, player_el] of this.player_els.entries()) {
        player_el.gameStarted(game, !game.game_base.game_ended && player_id === game.turn, !game.game_base.game_ended && player_id === game.dealer);
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
          this.setBackOfCard();
        } else {
          this.setTrumpImage();
        }
      } else {
        this.current_trick = game.teams.reduce((a, b) => a + b.tricks, 1);
        this.trick_number.innerText = this.current_trick.toString();
        this.status_container.innerText = `${this.game.players[this.game.turn].player.nickname} Playing`;
        this.setTrumpImage();
      }
    }
  }

  private setBackOfCard() {
    this.card_face_up_img.src = `/images/cards/card_back.png`;
    this.card_face_up_img.classList.add('show');
  }

  private setTrumpImage() {
    this.card_face_up_img.src = `/images/cards/suit_${cardSuitToName(this.game.trump_suit)}s.png`;
    this.card_face_up_img.classList.add('show');
  }

  async gameUpdate(update: UpdateMessage): Promise<void> {
    try {
      switch(update.kind) {
        case 'deal-round':
          const dealRoundData = update.update as DealRound;
          await this.applyDealRound(dealRoundData);
          break;
        case 'pass':
          const passData = update.update as PlayerPass;
          if (this.game.turn !== passData.player_id) {
            // TODO: try to sync data
            throw new Error('Player pass out of order');
          }
          if (this.game.bidding_choosing_trump && this.game.turn === this.game.dealer) {
            // TODO: try to sync data
            throw new Error('Dealer is stuck and must choose trump');
          }
          await this.applyPlayerPass(passData);
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
    const animation_time = !!data.cards ? Math.min(150 * data.cards.length, 4000) + 250 : 0;
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
    this.game.turn = (this.game.dealer + 1) % this.game.players.length;
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
    this.player_els[this.game.turn].bidding(false, false);
    this.status_container.innerText = `${this.game.players[this.game.turn].player.nickname} Bidding`;
  }

  private async applyPlayerPass(data: PlayerPass) {
    await this.player_els[data.player_id].setPassAnimation();
    this.player_els[data.player_id].setPass();
    if (this.game.turn === this.game.dealer) {
      this.game.bidding = false;
      this.game.bidding_choosing_trump = true;
      this.setBackOfCard(); // TODO: animation to flip over
    }
    this.game.turn = (this.game.turn + 1) % this.game.players.length;
    this.player_els[this.game.turn].bidding(this.game.bidding_choosing_trump, this.game.turn === this.game.dealer, this.game.card_face_up.suit);
    const bidding_text = this.game.bidding ? 'Bidding' : 'Choosing Trump';
    this.status_container.innerText = `${this.game.players[this.game.turn].player.nickname} ${bidding_text}`;
  }

  private async applyPlayerBid(data: PlayerBid) {
    await this.player_els[data.player_id].setBidAnimation(data.going_alone);
    this.game.bidding = false;
    this.game.makers_team = data.player_id % 2;
    this.game.defenders_team = this.game.makers_team == 0 ? 1 : 0;
    for (const player_id of this.game.teams[this.game.makers_team].player_ids) {
      const going_alone_ally = data.player_id !== player_id;
      this.player_els[player_id].setBid(true, data.going_alone, going_alone_ally);
      if (data.going_alone && going_alone_ally && this.player_id === player_id) {
        this.players_cards.removeCards();
      }
    }
    for (const player_id of this.game.teams[this.game.defenders_team].player_ids) {
      this.player_els[player_id].setBid(false);
    }
    this.game.player_bid = data.player_id;
    this.game.going_alone = data.going_alone;
    this.game.turn = this.game.dealer + 1;
    this.resolveTurn();
    this.game.trick_leader = this.game.turn;
    this.trick_number.innerText = '1';
    this.current_trick = 1;
    this.game.trump_suit = this.game.card_face_up.suit;
    if (data.going_alone) {
      this.setTrumpImage();
      this.setPlaying();
    } else {
      this.game.dealer_substituting_card = true;
      this.player_els[this.game.dealer].substitutingCard();
      this.status_container.innerText = `${this.game.players[this.game.dealer].player.nickname} Substituting Card`;
      if (this.game.dealer === this.player_id) {
        this.players_cards.can_play = true;
      }
    }
  }

  private async applyBidChooseTrump(data: BidChooseTrump) {
    await this.player_els[data.player_id].setBidAnimation(data.going_alone, cardSuitToName(data.trump_suit));
    this.game.bidding_choosing_trump = false;
    this.game.makers_team = data.player_id % 2;
    this.game.defenders_team = this.game.makers_team == 0 ? 1 : 0;
    for (const player_id of this.game.teams[this.game.makers_team].player_ids) {
      const going_alone_ally = data.player_id !== player_id;
      this.player_els[player_id].setBid(true, data.going_alone, going_alone_ally);
      if (data.going_alone && going_alone_ally && this.player_id === player_id) {
        this.players_cards.removeCards();
      }
    }
    for (const player_id of this.game.teams[this.game.defenders_team].player_ids) {
      this.player_els[player_id].setBid(false);
    }
    this.game.player_bid = data.player_id;
    this.game.going_alone = data.going_alone;
    this.game.turn = this.game.dealer + 1;
    this.resolveTurn();
    this.game.trick_leader = this.game.turn;
    this.trick_number.innerText = '1';
    this.current_trick = 1;
    this.game.trump_suit = data.trump_suit;
    this.setTrumpImage();
    this.setPlaying();
  }

  private async applyDealerSubstitutesCard(data: DealerSubstitutesCard) {
    this.game.dealer_substituting_card = false;
    this.player_els[data.player_id].substitutedCard();
    this.setTrumpImage();
    if (this.player_id === this.game.dealer) {
      this.players_cards.substituteCard(data.card_index, this.game.card_face_up);
    }
    for (const player_el of this.player_els.values()) {
      player_el.endBidding();
    }
    this.setPlaying();
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
    if (this.resolveTurn()) {
      this.game.trick.push({number: 0, suit: 0});
    }
    if (this.game.turn !== this.game.trick_leader) {
      this.setPlaying();
      return;
    }
    // end of trick
    await untilTimer(500);
    let winning_index = 0;
    let winning_card = this.game.trick[0];
    for (let i = 1; i < this.game.trick.length; i++) {
      const card = this.game.trick[i];
      if (this.cardSuit(card) === this.cardSuit(winning_card)) {
        if (this.cardNumber(card) > this.cardNumber(winning_card)) {
          winning_index = i;
          winning_card = card;
        }
      } else if (this.cardSuit(card) === this.game.trump_suit) {
        winning_index = i;
        winning_card = card;
      }
    }
    if (winning_index >= 0 && winning_index < this.trick_card_els.length) {
      this.trick_card_els[winning_index].style.zIndex = '2';
    }
    this.game.turn = (this.game.trick_leader + winning_index) % this.game.players.length;
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
    getPlayersTeam(this.game, this.game.turn).tricks++;
    for (const [i, player_el] of this.player_els.entries()) {
      player_el.endTrick(getPlayersTeam(this.game, i).tricks);
    }
    this.trick_cards.replaceChildren();
    this.trick_card_els = [];
    this.game.trick = [];
    this.trick_number.innerText = '-';
    console.log(`Trick won by ${this.game.players[this.game.turn].player.nickname} with the ${cardToName(winning_card)}`);
    if (this.current_trick < 5) {
      this.current_trick++;
      this.trick_number.innerText = this.current_trick.toString();
      this.setPlaying();
      return;
    }
    // end of round
    await untilTimer(500);
    // TODO: score animations for a few seconds
    this.card_face_up_img.classList.remove('show');
    let winning_team = 0;
    if (this.game.teams[0].tricks < 3) {
      winning_team = 1;
    }
    const won_all_five = this.game.teams[winning_team].tricks === 5;
    let game_over = false;
    if (winning_team === this.game.makers_team) {
      // makers won
      if (won_all_five) {
        if (this.game.going_alone) {
          game_over = this.scorePoints(this.game.makers_team, 4);
        } else {
          game_over = this.scorePoints(this.game.makers_team, 2);
        }
      } else {
        game_over = this.scorePoints(this.game.makers_team, 1);
      }
    } else {
      // defenders won
      game_over = this.scorePoints(this.game.defenders_team, 2);
    }
    for (const [i, _] of this.game.players.entries()) {
      this.player_els[i].endRound();
    }
    this.round_number.innerText = '-';
    if (game_over) {
      this.game.game_base.game_ended = true;
      this.status_container.innerText = 'game over';
    } else {
      this.game.round++; // wait for deal-round update from server
    }
  }

  private setPlaying() {
    this.player_els[this.game.turn].playing();
    this.status_container.innerText = `${this.game.players[this.game.turn].player.nickname} Playing`;
    if (this.game.turn === this.player_id) {
      this.players_cards.can_play = true;
    }
  }

  private setCardFaceUpImage(card: StandardCard) {
    this.card_face_up_img.src = cardToImagePath(card);
    this.card_face_up_img.classList.add('show');
  }

  private resolveTurn(): boolean {
    if (this.game.turn >= this.game.players.length) {
      this.game.turn -= this.game.players.length;
    }
    if (this.game.going_alone && (Math.round(Math.abs(this.game.turn - this.game.player_bid)) === 2)) {
      this.game.turn++;
      if (this.game.turn >= this.game.players.length) {
        this.game.turn -= this.game.players.length;
      }
      return true;
    }
    return false;
  }

  private cardSuit(card: StandardCard): number {
    if ((cardSuitToColor(card.suit) === cardSuitToColor(this.game.trump_suit)) && card.number == 11) {
      return this.game.trump_suit;
    }
    return card.suit;
  }

  private cardNumber(card: StandardCard): number {
    if (card.suit == this.game.trump_suit && card.number == 11) {
      // right bar
      return 16;
    } else if (this.cardSuit(card) == this.game.trump_suit && card.number == 11) {
      // left bar
      return 15;
    }
    return card.number;
  }

  private scorePoints(team_id: number, points: number): boolean {
    this.game.teams[team_id].score += points;
    for (const winner of this.game.teams[team_id].player_ids) {
      this.player_els[winner].setScore(this.game.teams[team_id].score);
    }
    if (this.game.teams[team_id].score >= 10) {
      const winners: string[] = [];
      for (const winner of this.game.teams[team_id].player_ids) {
        this.player_els[winner].wonGame();
        winners.push(this.game.players[winner].player.nickname);
      }
      let winner_text = 'The winners are: ' + winners.join(', ');
      winner_text += `\nWith ${this.game.teams[team_id].score} points`;
      messageDialog.call(this, {message: winner_text});
      return true;
    }
    return false;
  }
}

customElements.define('dwg-euchre', DwgEuchre);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-euchre': DwgEuchre;
  }
}
