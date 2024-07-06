import { DwgElement } from '../../../../dwg_element';
import { EuchrePlayer, EuchreTeam, GameEuchre } from '../euchre_data';
import './euchre_player.scss';
export declare class DwgEuchrePlayer extends DwgElement {
    name_container: HTMLDivElement;
    status_container: HTMLDivElement;
    score_container: HTMLSpanElement;
    tricks_container: HTMLSpanElement;
    icons_wrapper: HTMLDivElement;
    bid_animation: HTMLDivElement;
    bid_input_wrapper: HTMLDivElement;
    bid_input_message: HTMLDivElement;
    going_alone: HTMLInputElement;
    bid_button: HTMLButtonElement;
    spades_button: HTMLButtonElement;
    diamonds_button: HTMLButtonElement;
    clubs_button: HTMLButtonElement;
    hearts_button: HTMLButtonElement;
    pass_button: HTMLButtonElement;
    initialized: boolean;
    player: EuchrePlayer;
    team: EuchreTeam;
    client_player: boolean;
    dealer: boolean;
    icons: Map<string, HTMLDivElement>;
    constructor();
    protected parsedCallback(): void;
    initialize(player: EuchrePlayer, team: EuchreTeam): void;
    gameStarted(game: GameEuchre, current_turn: boolean, dealer: boolean): Promise<void>;
    setClientPlayer(): void;
    private disableButtons;
    private toggleIcon;
    newRound(dealer: boolean): void;
    endRound(): void;
    setDealer(dealer: boolean): void;
    bidding(bidding_choosing_trump: boolean, is_dealer: boolean, card_face_up_suit?: number): void;
    setPassAnimation(): Promise<void>;
    setPass(): void;
    setBidAnimation(going_alone: boolean, trump_suit_name?: string): Promise<void>;
    setBid(makers: boolean, going_alone?: boolean, going_alone_ally?: boolean): void;
    substitutingCard(): void;
    substitutedCard(): void;
    endBidding(): void;
    playing(): void;
    playCard(): void;
    endTrick(tricks: number): void;
    setScore(score: number): void;
    wonGame(): void;
}
declare global {
    interface HTMLElementTagNameMap {
        'dwg-euchre-player': DwgEuchrePlayer;
    }
}
