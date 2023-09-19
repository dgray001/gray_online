import {GameType} from "../lobby/data_models";

/** Data describing a game */
export declare interface Game {
  game_base: GameBase;
  // game specific object
}

/** Data describing a game returned from the server */
export declare interface GameFromServer {
  game_base: GameBaseGameBaseFromServer;
  // game specific object
}

/** Converts a GameFromServer to a proper frontend game object */
export function serverResponseToGame(game: GameFromServer): Game {
  return {
    game_base: {
      game_id: game.game_base.game_id,
      game_type: game.game_base.game_type,
      game_started: game.game_base.game_started,
      game_ended: game.game_base.game_ended,
      players: new Map(game.game_base.players.map(player => [player.client_id, player])),
      viewers: new Map(game.game_base.viewers.map(viewer => [viewer.client_id, viewer])),
    },
  }
}

/** Data describing a game base */
export declare interface GameBaseGameBaseFromServer {
  game_id: number;
  game_type: GameType;
  game_started: boolean;
  game_ended: boolean;
  players: GamePlayer[];
  viewers: GameViewer[];
}

/** Data describing a game base */
export declare interface GameBase {
  game_id: number;
  game_type: GameType;
  game_started: boolean;
  game_ended: boolean;
  players: Map<number, GamePlayer>;
  viewers: Map<number, GameViewer>;
}

/** Data describing a game player */
export declare interface GamePlayer {
  client_id: number;
  nickname: string;
  connected: boolean;
}

/** Data describing a game viewer */
export declare interface GameViewer {
  client_id: number;
  nickname: string;
  connected: boolean;
}