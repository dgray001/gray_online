import type { ConnectData } from './components/lobby/lobby_connector/lobby_connector';
import type { LobbyRoom } from './components/lobby/data_models';
import type { MessageDialogData } from './components/dialog_box/message_dialog/message_dialog';
import type { ChatMessage } from './components/chatbox/chatbox';
import type { JoinRoomData } from './components/lobby/lobby_rooms/lobby_rooms';
import type { CanvasBoardSize } from './components/game/util/canvas_board/canvas_board';

// Custom events go here
interface MyCustomEventMap {
  connect: CustomEvent<ConnectData>;
  game_launched: CustomEvent<LobbyRoom>;
  rejoin_game: CustomEvent<LobbyRoom>;
  refresh_game_lobby: CustomEvent<LobbyRoom>;
  show_message_dialog: CustomEvent<MessageDialogData>;
  connection_lost: CustomEvent<string>;
  chat_sent: CustomEvent<ChatMessage>;
  game_update: CustomEvent<string>;
  join_room: CustomEvent<JoinRoomData>;
  rename_room: CustomEvent<string>;
  kick_player: CustomEvent<number>;
  viewer_player: CustomEvent<number>;
  player_player: CustomEvent<number>;
  promote_player: CustomEvent<number>;
  save_settings: CustomEvent<string>;
  join_lobby_room: CustomEvent<boolean>;
  play_card: CustomEvent<number>;
  canvas_resize: CustomEvent<CanvasBoardSize>;
}

declare global {
  interface Headers {
    'X-File-Name': string;
  }

  interface WindowEventMap extends MyCustomEventMap {}
  interface HTMLElementEventMap extends MyCustomEventMap {}
  interface DocumentEventMap extends MyCustomEventMap {}
}

export {};
