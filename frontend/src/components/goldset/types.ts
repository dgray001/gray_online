export interface GoldCard {
  suit: number;
  number: number;
  index: number;
  action: number;
  played: boolean;
}
export interface GoldPlayer {
  seat: number;
  score: number;
  bid: number;
  has_bid: boolean;
  tricks: number;
  voids: boolean[];
  profile: Record<string, number> | null;
  prior_profile: Record<string, number> | null;
}
export interface GoldEvent {
  seat: number;
  round: number;
  hand: number;
  trick: number;
  order: number;
  kind: string;
  suit: number;
  number: number;
  bid: number;
}
export interface GoldState {
  kind: 'bet' | 'play-card';
  objective: 'win';
  seat: number;
  dealer: number;
  trick_leader: number;
  hand_size: number;
  max_hand: number;
  increasing: boolean;
  progress: { index: number; total: number; remaining: number; future_points: number };
  round_points: number;
  trick_points: number;
  trump: GoldCard;
  hand: GoldCard[];
  trick: GoldCard[] | null;
  players: GoldPlayer[];
  history: GoldEvent[] | null;
}
export interface ExplainedCard {
  index: number;
  suit: number;
  number: number;
  win_probability: number;
  contribution: number;
  utility: number;
  heuristic: number;
  probability: number;
}
export interface GoldAnalysis {
  kind: string;
  raw_bid: number;
  bid_probabilities: number[] | null;
  trick_probabilities: number[] | null;
  cards: ExplainedCard[];
  assumption: string;
  unknown_cards: number;
  beating_cards: number;
  scale: number;
  optimism: number;
  temperature: number;
}
export interface GoldLabel {
  acceptable_actions?: number[];
  item_id: string;
  action: number;
  notes: string;
  confidence: number;
  saved_at: string;
}
export interface GoldItem {
  created_at: string;
  id: string;
  state: GoldState;
  legal: number[];
  analysis: GoldAnalysis;
  model4_action: number;
  label: GoldLabel | null;
}
export interface QueueItem {
  id: string;
  kind: string;
  players: number;
  hand_size: number;
  seat: number;
  labelled: boolean;
}
