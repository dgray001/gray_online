
/** Data describing fiddlesticks-specific game settings */
export declare interface GameSettingsFiddlesticks {
  min_round?: number;
  max_round?: number;
  round_points: number;
  trick_points: number;
  ai_players: AiPlayerFiddlesticks[];
}

/** Deta describing the settings for an ai player in fiddlesticks */
export declare interface AiPlayerFiddlesticks {
  nickname: string;
  //model: AiPlayerFiddlesticksModel;
}