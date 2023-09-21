/** Data describing a standard card */
export declare interface StandardCard {
  suit: number;
  number: number;
}

/** Maps suits to their string name */
function cardSuitToName(suit: number): string {
  switch(suit) {
    case 1:
      return "Diamond";
    case 2:
      return "Club";
    case 3:
      return "Heart";
    case 4:
      return "Spade";
    default:
      return "error";
  }
}

/** Maps numbers to their string name */
function cardNumberToName(card_number: number): string {
  if (card_number > 1 && card_number < 11) {
    return card_number.toString();
  }
  switch(card_number) {
    case 11:
      return "jack";
    case 12:
      return "queen";
    case 13:
      return "king";
    case 14:
      return "ace";
    default:
      return "error";
  }
}

/** Returns string name of card */
export function cardToName(card: StandardCard): string {
  return `${cardNumberToName(card.number)} of ${cardSuitToName(card.suit)}s`;
}

/** Returns path ready to insert into src for a standard card */
export function cardToImagePath(card: StandardCard): string {
  let return_string = `${cardNumberToName(card.number)}_of_${cardSuitToName(card.suit)}s`;
  if (card.number > 10 && card.number < 14) {
    return_string += '2';
  }
  return `/cards/${return_string}.png`;
}