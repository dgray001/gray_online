import {DwgElement} from '../dwg_element';
import {ServerMessage, LobbyRoom, LobbyGameFromServer} from '../lobby/data_models';
import {apiGet} from '../../scripts/api';

import {handleMessage} from './message_handler';
import html from './game.html';
import './game.scss';

export class DwgGame extends DwgElement {
  client_name_string: HTMLSpanElement;
  client_ping: HTMLSpanElement;
  room_name: HTMLDivElement;
  game_name: HTMLDivElement;
  waiting_room: HTMLDivElement;
  players_waiting: HTMLDivElement;
  game_container: HTMLDivElement;

  launched: boolean;
  socket: WebSocket;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('client_name_string');
    this.configureElement('client_ping');
    this.configureElement('room_name');
    this.configureElement('game_name');
    this.configureElement('waiting_room');
    this.configureElement('players_waiting');
    this.configureElement('game_container');
  }

  protected override parsedCallback(): void {
  }

  async launchGame(lobby: LobbyRoom, socket: WebSocket) {
    if (this.launched || !lobby || !lobby.game_id || !socket || socket.readyState !== WebSocket.OPEN) {
      return;
    }
    try {
      const response = await apiGet<LobbyGameFromServer>(`lobby/games/get/${lobby.game_id}`);
      if (response.success) {
        console.log(response.result);
        this.classList.add('show');
        this.socket = socket;
        this.socket.addEventListener('message', (m) => {
          try {
            const message = JSON.parse(m.data) as ServerMessage;
            handleMessage(this, message);
          } catch(e) {
            console.log("Error parsing message: ", m, e)
          }
        });
        // TODO: tell server you're connected
      } else {
        console.log(response.error_message);
      }
    } catch(e) {}
  }

  socketActive() {
    return !!this.socket && this.socket.readyState == WebSocket.OPEN;
  }
}

customElements.define('dwg-game', DwgGame);
