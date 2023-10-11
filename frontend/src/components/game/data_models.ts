import {GameType, ServerMessage} from "../lobby/data_models";

/** Interface for components that represent a frontend game */
export declare interface GameComponent {
  initialize(game: Game, client_id: number): void;
  gameUpdate(update: UpdateMessage): void;
}

/** Data describing a game */
export declare interface Game {
  game_base: GameBase;
  // ... game specific fields
}

/** Update message for clients (sent by server) */
export declare interface UpdateMessage {
  update_id: number;
  kind: string;
  update: any; // type needs to be coerced by game logic
}

/** Update message for server (sent by client) */
export declare interface PlayerAction {
  action_id: number;
  client_id: number;
  kind: string;
  action: any; // type needs to be coerced by game logic
}

/** Data describing a game returned from the server */
export declare interface GameFromServer {
  game_base: GameBaseFromServer;
  // ... game specific fields
}

/** Converts a GameFromServer to a proper frontend game object */
export function serverResponseToGame(gameFromServer: GameFromServer, client_id: number): Game {
  const players = new Map(gameFromServer.game_base.players.map(player => [player.client_id, player]));
  const viewers = new Map(gameFromServer.game_base.viewers.map(viewer => [viewer.client_id, viewer]));
  const game = {
    ...gameFromServer, // ... game specific fields
    game_base: {
      game_id: gameFromServer.game_base.game_id,
      game_type: gameFromServer.game_base.game_type,
      game_started: gameFromServer.game_base.game_started,
      game_ended: gameFromServer.game_base.game_ended,
      players,
      viewers,
      player_actions: gameFromServer.game_base.player_actions ?
        new Map(gameFromServer.game_base.player_actions.map(action => [action.action_id, action])) :
        undefined,
    },
  } as Game;
  const updates = players.get(client_id)?.updates ?? viewers.get(client_id)?.updates;
  if (updates !== undefined) {
    game.game_base.updates = new Map(updates.map(update => [update.update_id, update]));
    game.game_base.last_continuous_update_id = updates.length;
  }
  return game;
}

/** Data describing a game base */
export declare interface GameBaseFromServer {
  game_id: number;
  game_type: GameType;
  game_started: boolean;
  game_ended: boolean;
  players: GamePlayer[];
  viewers: GameViewer[];
  player_actions?: PlayerAction[];
}

/** Data describing a game base */
export declare interface GameBase {
  game_id: number;
  game_type: GameType;
  game_started: boolean;
  game_ended: boolean;
  players: Map<number, GamePlayer>;
  viewers: Map<number, GameViewer>;
  player_actions?: Map<number, PlayerAction>; // client -> server updates for viewers
  // TODO: add last_continuous_action_id for viewer support
  updates?: Map<number, UpdateMessage>; // server -> client updates for players
  last_continuous_update_id?: number; // to keep track of which updates are needed
}

/** Data describing a game player */
export declare interface GamePlayer {
  client_id: number;
  player_id: number;
  nickname: string;
  connected: boolean;
  updates?: UpdateMessage[];
}

/** Data describing a game viewer */
export declare interface GameViewer {
  client_id: number;
  nickname: string;
  connected: boolean;
  updates?: UpdateMessage[];
}