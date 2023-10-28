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

/** Maps suits to their icon */
function cardSuitToIcon(suit: number): string {
  switch(suit) {
    case 1:
      return "♦";
    case 2:
      return "♣";
    case 3:
      return "♥";
    case 4:
      return "♠";
    default:
      return "✪";
  }
}

/** Maps numbers to their string name */
function cardNumberToName(card_number: number): string {
  if (card_number > 1 && card_number < 11) {
    return card_number.toString();
  }
  switch(card_number) {
    case 11:
      return "Jack";
    case 12:
      return "Queen";
    case 13:
      return "King";
    case 14:
      return "Ace";
    default:
      return "error";
  }
}

/** Maps numbers to their icon name */
function cardNumberToIconName(card_number: number): string {
  if (card_number > 1 && card_number < 11) {
    return card_number.toString();
  }
  switch(card_number) {
    case 11:
      return "J";
    case 12:
      return "Q";
    case 13:
      return "K";
    case 14:
      return "A";
    default:
      return "?";
  }
}

/** Returns string name of card */
export function cardToName(card: StandardCard): string {
  return `${cardNumberToName(card.number)} of ${cardSuitToName(card.suit)}s`;
}

/** Returns icon name of card */
export function cardToIcon(card: StandardCard, render_html = true): string {
  if (render_html) {
    const color = (card.suit === 1 || card.suit === 3) ? '#df0000' : 'black';
    return `<span style="color:${color};">${cardNumberToIconName(card.number)}<span style="font-size:1.4rem;">${cardSuitToIcon(card.suit)}</span></span>`;
  } else {
    return `${cardNumberToIconName(card.number)}${cardSuitToIcon(card.suit)}`;
  }
}

/** Returns path ready to insert into src for a standard card */
export function cardToImagePath(card: StandardCard): string {
  let return_string = `${cardNumberToName(card.number).toLowerCase()}_of_${cardSuitToName(card.suit).toLowerCase()}s`;
  if (card.number > 10 && card.number < 14) {
    return_string += '2';
  }
  return `/cards/${return_string}.png`;
}