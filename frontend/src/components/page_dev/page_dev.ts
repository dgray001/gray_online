import { DwgElement } from '../dwg_element';
import type { DwgGame } from '../game/game';
import { getUrlParam } from '../../scripts/url';
import { websocketPath } from '../../scripts/api';
import type { GameTypeLowerKeys, ConnectionMetadata, GameSettings, LobbyRoom } from '../lobby/data_models';
import {
  createMessage,
  GameType,
  defaultBaseGameSettings,
  getGameTypeFromLowercaseString,
  isValidGameTypeString,
} from '../lobby/data_models';
import type { DwgLobby } from '../lobby/lobby';
import { until } from '../../scripts/util';

import html from './page_dev.html';

import './page_dev.scss';

export class DwgPageDev extends DwgElement {
  private game!: DwgGame;
  private lobby!: DwgLobby;

  constructor() {
    super();
    this.html_string = html;
    this.configureElements('game', 'lobby');
  }

  protected override parsedCallback(): void {
    const launch_game_param = getUrlParam('launchgame')?.toLowerCase();
    if (launch_game_param) {
      if (!isValidGameTypeString(launch_game_param)) {
        console.error(`Launch game parameter is an invalid game type: ${launch_game_param}`);
        return;
      }
      this.launchGame(launch_game_param);
      return;
    }
  }

  private launchGame(game: GameTypeLowerKeys) {
    const nickname = 'dev_user';
    const socket = new WebSocket(`${websocketPath()}/connect/${nickname}`);
    socket.addEventListener('error', (e) => {
      console.error(e);
    });
    socket.addEventListener('open', async () => {
      this.lobby.connect(nickname, socket);
      this.game.exitGame();
      this.lobby.exitGame();
      this.lobby.classList.add('hide');
      let connection_metadata!: ConnectionMetadata;
      await until(() => {
        connection_metadata = this.lobby.getConnectionMetadata();
        return !!connection_metadata;
      });
      socket.send(createMessage(`client-${connection_metadata.client_id}`, 'room-create'));
      let lobby_room!: LobbyRoom;
      await until(() => {
        lobby_room = this.lobby.getLobbyRoom().getRoom() as LobbyRoom;
        return !!lobby_room;
      });
      const game_type = getGameTypeFromLowercaseString(game);
      const game_settings = this.createGameSettings(game_type);
      socket.send(
        createMessage(
          `client-${connection_metadata.client_id}`,
          'room-settings-update',
          JSON.stringify(game_settings),
          lobby_room.room_id.toString()
        )
      );
      await until(() => this.lobby.getLobbyRoom().getRoom()?.game_settings.game_type === game_type);
      socket.send(
        createMessage(`client-${connection_metadata.client_id}`, 'room-launch', '', lobby_room.room_id.toString())
      );
      await until(() => !!this.lobby.getLobbyRoom().getRoom()?.game_id);
      console.log('Launching game from dev page with the background room:', lobby_room);
      this.game.launchGame(lobby_room, this.lobby.getSocket(), connection_metadata);
    });
    this.lobby.addEventListener('connection_lost', () => {
      console.error('Dev background lobby lost connection');
    });
  }

  private createGameSettings(game_type: GameType): GameSettings {
    const base = defaultBaseGameSettings();
    switch (game_type) {
      case GameType.FIDDLESTICKS:
        return {
          ...base,
          game_type,
          game_specific_settings: {
            round_points: 10,
            trick_points: 1,
            ai_players: [],
          },
        };
      default:
        return {
          ...base,
          game_type,
        };
    }
  }
}

customElements.define('dwg-page-dev', DwgPageDev);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-page-dev': DwgPageDev;
  }
}
