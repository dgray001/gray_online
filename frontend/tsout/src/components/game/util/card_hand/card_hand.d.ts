import { DwgElement } from '../../../dwg_element';
import { StandardCard } from '../card_util';
import './card_hand.scss';
export declare class DwgCardHand extends DwgElement {
    private cards_container;
    private play_drop;
    private cancel_drop;
    can_play: boolean;
    private cards;
    private dragging_data;
    private dragging_lock;
    constructor();
    protected parsedCallback(): Promise<void>;
    private resizeCallback;
    private setEventListeners;
    setCards(cards: StandardCard[], cards_played?: number[], animation_time?: number): void;
    private createCardEl;
    private startDraggingCard;
    private dragCard;
    private stopDragging;
    private stopDraggingCard;
    playCard(index: number): void;
    substituteCard(index: number, new_card: StandardCard): void;
    removeCards(): void;
}
declare global {
    interface HTMLElementTagNameMap {
        'dwg-card-hand': DwgCardHand;
    }
}
