export declare interface StandardCard {
    suit: number;
    number: number;
}
export declare function cardSuitToColor(suit: number): string;
export declare function cardSuitToName(suit: number): string;
export declare function cardToName(card: StandardCard): string;
export declare function cardToIcon(card: StandardCard, render_html?: boolean): string;
export declare function cardToImagePath(card: StandardCard): string;
