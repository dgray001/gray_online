import { DwgElement } from '../../../../dwg_element';
import { FiddlesticksPlayer } from '../fiddlesticks_data';
import './fiddlesticks_player.scss';
export declare class DwgFiddlesticksPlayer extends DwgElement {
    name_container: HTMLDivElement;
    status_container: HTMLDivElement;
    score_container: HTMLSpanElement;
    bet_container: HTMLSpanElement;
    bet_input_wrapper: HTMLDivElement;
    bet_input: HTMLInputElement;
    bet_button: HTMLButtonElement;
    tricks_container: HTMLSpanElement;
    dealer_wrapper: HTMLDivElement;
    winner_wrapper: HTMLDivElement;
    bet_animation: HTMLDivElement;
    initialized: boolean;
    player: FiddlesticksPlayer;
    client_player: boolean;
    card_els: HTMLDivElement[];
    constructor();
    protected parsedCallback(): void;
    initialize(player: FiddlesticksPlayer): void;
    gameStarted(betting: boolean, current_turn: boolean, dealer: boolean): Promise<void>;
    setClientPlayer(): void;
    sendBetEvent(): void;
    newRound(dealer: boolean): void;
    setDealer(dealer: boolean): void;
    endRound(): void;
    betting(): void;
    setBetAnimation(amount: number): Promise<void>;
    setBet(amount: number): void;
    endBetting(): void;
    playing(): void;
    playCard(): void;
    endTrick(tricks: number): void;
    setScore(score: number): void;
    wonGame(): void;
}
declare global {
    interface HTMLElementTagNameMap {
        'dwg-fiddlesticks-player': DwgFiddlesticksPlayer;
    }
}
