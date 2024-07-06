import { DwgElement } from '../../../dwg_element';
import { GameComponent, UpdateMessage } from '../../data_models';
import { DwgCardHand } from '../../util/card_hand/card_hand';
import { DwgGame } from '../../game';
import { DwgEuchrePlayer } from './euchre_player/euchre_player';
import { GameEuchre } from './euchre_data';
import './euchre.scss';
import './euchre_player/euchre_player';
import '../../../dialog_box/message_dialog/message_dialog';
import '../../util/card_hand/card_hand';
export declare class DwgEuchre extends DwgElement implements GameComponent {
    round_number: HTMLSpanElement;
    trick_number: HTMLSpanElement;
    status_container: HTMLSpanElement;
    card_face_up_img: HTMLImageElement;
    table_container: HTMLDivElement;
    trick_cards: HTMLDivElement;
    player_container: HTMLDivElement;
    players_cards: DwgCardHand;
    game: GameEuchre;
    current_trick: number;
    player_els: DwgEuchrePlayer[];
    player_id: number;
    trick_card_els: HTMLDivElement[];
    constructor();
    protected parsedCallback(): void;
    initialize(abstract_game: DwgGame, game: GameEuchre): Promise<void>;
    private setBackOfCard;
    private setTrumpImage;
    gameUpdate(update: UpdateMessage): Promise<void>;
    private applyDealRound;
    private applyPlayerPass;
    private applyPlayerBid;
    private applyBidChooseTrump;
    private applyDealerSubstitutesCard;
    private applyPlayCard;
    private setPlaying;
    private setCardFaceUpImage;
    private resolveTurn;
    private cardSuit;
    private cardNumber;
    private scorePoints;
}
declare global {
    interface HTMLElementTagNameMap {
        'dwg-euchre': DwgEuchre;
    }
}
