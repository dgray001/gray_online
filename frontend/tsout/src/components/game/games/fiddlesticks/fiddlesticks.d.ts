import { DwgElement } from '../../../dwg_element';
import { GameComponent, UpdateMessage } from '../../data_models';
import { DwgGame } from '../../game';
import { GameFiddlesticks } from './fiddlesticks_data';
import './fiddlesticks.scss';
import './fiddlesticks_player/fiddlesticks_player';
import '../../../dialog_box/message_dialog/message_dialog';
import '../../util/card_hand/card_hand';
export declare class DwgFiddlesticks extends DwgElement implements GameComponent {
    private round_number;
    private bets_number;
    private trick_number;
    private status_container;
    private trump_card_img;
    private trick_cards;
    private table_container;
    private player_container;
    private players_cards;
    private game;
    private current_trick;
    private player_els;
    private player_id;
    private trick_card_els;
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
