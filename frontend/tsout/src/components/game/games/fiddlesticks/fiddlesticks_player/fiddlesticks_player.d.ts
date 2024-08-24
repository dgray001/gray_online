import { DwgElement } from '../../../../dwg_element';
import { FiddlesticksPlayer } from '../fiddlesticks_data';
import './fiddlesticks_player.scss';
export declare class DwgFiddlesticksPlayer extends DwgElement {
    private name_container;
    private status_container;
    private score_container;
    private bet_container;
    private bet_input_wrapper;
    private bet_input;
    private bet_button;
    private tricks_container;
    private dealer_wrapper;
    private winner_wrapper;
    private bet_animation;
    private initialized;
    private player;
    private client_player;
    private card_els;
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
