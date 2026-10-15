import type { UpdateMessage } from '../../../data_models';
import type { DealRound, PlayCard, PlayerBet } from '../fiddlesticks_data';

export interface HistoryTrick {
  plays: PlayCard[];
  winner?: PlayCard;
}

export interface HistoryRound {
  deal: DealRound;
  bets: Map<number, number>;
  tricks: HistoryTrick[];
  scores: HistoryScore[];
}

export interface HistoryScore {
  bet?: number;
  won: number;
  points?: number;
  total: number;
}

export function winningPlay(plays: readonly PlayCard[], trump_suit: number): PlayCard {
  return plays.reduce((winner: PlayCard, play: PlayCard): PlayCard => {
    const same_suit = play.card.suit === winner.card.suit;
    return (same_suit && play.card.number > winner.card.number) || (!same_suit && play.card.suit === trump_suit)
      ? play
      : winner;
  });
}

export function historyRounds(
  updates: readonly UpdateMessage[],
  player_count: number,
  round_points: number,
  trick_points: number
): HistoryRound[] {
  const rounds: HistoryRound[] = [];
  const ordered = [...updates].sort((a: UpdateMessage, b: UpdateMessage): number => a.update_id - b.update_id);
  for (const update of ordered) {
    if (update.kind === 'deal-round') {
      rounds.push({ deal: update.content as DealRound, bets: new Map(), tricks: [], scores: [] });
    } else if (rounds.length > 0) {
      const round = rounds[rounds.length - 1];
      if (update.kind === 'bet') {
        const bet = update.content as PlayerBet;
        round.bets.set(bet.player_id, bet.amount);
      } else if (update.kind === 'play-card') {
        appendPlay(round, update.content as PlayCard, player_count);
      }
    }
  }
  scoreRounds(rounds, player_count, round_points, trick_points);
  return rounds;
}

function appendPlay(round: HistoryRound, play: PlayCard, player_count: number): void {
  let trick = round.tricks[round.tricks.length - 1];
  if (!trick || trick.winner) {
    trick = { plays: [] };
    round.tricks.push(trick);
  }
  trick.plays.push(play);
  if (trick.plays.length === player_count) {
    trick.winner = winningPlay(trick.plays, round.deal.trump.suit);
  }
}

function scoreRounds(rounds: HistoryRound[], player_count: number, round_points: number, trick_points: number): void {
  const totals: number[] = Array<number>(player_count).fill(0);
  for (const round of rounds) {
    const wins: number[] = Array<number>(player_count).fill(0);
    for (const trick of round.tricks) {
      if (trick.winner) {
        wins[trick.winner.player_id]++;
      }
    }
    const complete = round.tricks.filter((trick: HistoryTrick): boolean => !!trick.winner).length === round.deal.round;
    round.scores = wins.map((won: number, id: number): HistoryScore => {
      const bet = round.bets.get(id);
      const points = complete ? (bet === won ? round_points + trick_points * won : 0) : undefined;
      totals[id] += points ?? 0;
      return { bet, won, points, total: totals[id] };
    });
  }
}
