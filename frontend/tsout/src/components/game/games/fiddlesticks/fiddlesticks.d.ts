import { DwgElement } from '../../../dwg_element';
import { GameComponent, UpdateMessage } from '../../data_models';
import { DwgFiddlesticksPlayer } from './fiddlesticks_player/fiddlesticks_player';
import { DwgCardHand } from '../../util/card_hand/card_hand';
import { DwgGame } from '../../game';
import { GameFiddlesticks } from './fiddlesticks_data';
import './fiddlesticks.scss';
import './fiddlesticks_player/fiddlesticks_player';
import '../../../dialog_box/message_dialog/message_dialog';
import '../../util/card_hand/card_hand';
export declare class DwgFiddlesticks extends DwgElement implements GameComponent {
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
    current_trick: number;
    player_els: DwgFiddlesticksPlayer[];
    player_id: number;
    trick_card_els: HTMLDivElement[];
    constructor();
    protected parsedCallback(): void;
    initialize(abstract_game: DwgGame, game: GameFiddlesticks): Promise<void>;
    private updateBetsContainer;
    private setTrumpImage;
    gameUpdate(update: UpdateMessage): Promise<void>;
    private applyDealRound;
    private applyBet;
    private applyPlayCard;
    private addPlayedCard;
}
declare global {
    interface HTMLElementTagNameMap {
        'dwg-fiddlesticks': DwgFiddlesticks;
    }
}
