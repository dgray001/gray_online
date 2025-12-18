import type { GameType } from '../lobby/data_models';
import type { DwgGame } from './game';

/** HTML tags for all game types */
export type GameHtmlTag = 'dwg-fiddlesticks' | 'dwg-euchre' | 'dwg-risq' | 'dwg-test-game';

/** Interface for components that represent a frontend game */
export declare interface GameComponent {
  initialize(abstract_game: DwgGame, game: Game): Promise<void>;
  gameUpdate(update: UpdateMessage): Promise<void>;
  updateDialogComponent(update: UpdateMessage): HTMLElement;
}

/** Data describing a game */
export declare interface Game {
  game_base: GameBase;
  // ... game specific fields
  players: {
    player: GamePlayer;
  }[];
}

/** Update message for clients (sent by server) */
export declare interface UpdateMessage {
  update_id: number;
  kind: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  content: any; // type needs to be coerced by game logic
}

/** Update message for server (sent by client) */
export declare interface PlayerAction {
  action_id: number;
  client_id: number;
  kind: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  action: any; // type needs to be coerced by game logic
}

/** Data describing a game returned from the server */
export declare interface GameFromServer {
  game_base: GameBaseFromServer;
  // ... game specific fields
}

/** Converts a GameFromServer to a proper frontend game object */
export function serverResponseToGame(game_from_server: GameFromServer, client_id: number): Game {
  const players = new Map(game_from_server.game_base.players.map((player) => [player.client_id, player]));
  const viewers = new Map(game_from_server.game_base.viewers.map((viewer) => [viewer.client_id, viewer]));
  const game = {
    ...game_from_server, // ... game specific fields
    game_base: {
      game_id: game_from_server.game_base.game_id,
      game_type: game_from_server.game_base.game_type,
      game_started: game_from_server.game_base.game_started,
      game_ended: game_from_server.game_base.game_ended,
      players,
      viewers,
      player_actions: game_from_server.game_base.player_actions
        ? new Map(game_from_server.game_base.player_actions.map((action) => [action.action_id, action]))
        : undefined,
    },
  } as Game;
  const updates = players.get(client_id)?.updates ?? game_from_server.game_base.viewer_updates;
  if (updates !== undefined) {
    game.game_base.updates = new Map(updates.map((update) => [update.update_id, update]));
    game.game_base.last_applied_update_id = updates.length;
    game.game_base.highest_received_update_id = Math.max(...updates.map((update) => update.update_id));
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
  // These fields should only return to viewers
  player_actions?: PlayerAction[];
  viewer_updates?: UpdateMessage[];
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
  updates?: Map<number, UpdateMessage>; // server -> client updates for players
  last_applied_update_id?: number; // to keep track of which updates are needed
  highest_received_update_id?: number; // to keep track of which updates are needed
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
