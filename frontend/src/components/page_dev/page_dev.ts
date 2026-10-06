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
import '../lobby/lobby_game_settings/game_specific_data';

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
    this.game.addEventListener('show_message_dialog', (e) => {
      const dialog = document.createElement('dwg-message-dialog');
      dialog.setData(e.detail);
      this.appendChild(dialog);
    });
    const launch_game_param = getUrlParam('launchgame')?.toLowerCase();
    if (launch_game_param) {
      if (!isValidGameTypeString(launch_game_param)) {
        console.error(`Launch game parameter is an invalid game type: ${launch_game_param}`);
        return;
      }
      let custom_game_settings = undefined;
      const settings_param = getUrlParam('game_settings');
      if (settings_param) {
        try {
          custom_game_settings = JSON.parse(settings_param);
        } catch (e) {
          console.error(`Invalid JSON in game_settings parameter: ${settings_param}`, e);
        }
      }
      this.launchGame(launch_game_param as GameTypeLowerKeys, custom_game_settings);
      return;
    }
  }

  private launchGame(game: GameTypeLowerKeys, custom_settings?: Record<string, unknown>) {
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
      const game_settings = this.createGameSettings(game_type, custom_settings);
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

  private createGameSettings(game_type: GameType, custom_settings?: Record<string, unknown>): GameSettings {
    const base = defaultBaseGameSettings();
    let settings: GameSettings;
    switch (game_type) {
      case GameType.FIDDLESTICKS:
        settings = {
          ...base,
          game_type,
          game_specific_settings: {
            round_points: 10,
            trick_points: 1,
            ai_players: [{ nickname: 'AI Player' }],
          },
        };
        break;
      case GameType.EGYPTIAN_RAT_CRAP:
        settings = {
          ...base,
          game_type,
          game_specific_settings: {
            ai_players: [{ nickname: 'AI Player' }],
          },
        };
        break;
      case GameType.RISQ:
        settings = {
          ...base,
          game_type,
          game_specific_settings: {
            ai_players: [{ nickname: 'AI Player 1' }, { nickname: 'AI Player 2' }],
          },
        };
        break;
      default:
        settings = {
          ...base,
          game_type,
        };
        break;
    }
    if (custom_settings) {
      settings.game_specific_settings = {
        ...settings.game_specific_settings,
        ...custom_settings,
      } as GameSettings['game_specific_settings'];
    }
    return settings;
  }
}

customElements.define('dwg-page-dev', DwgPageDev);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-page-dev': DwgPageDev;
  }
}
