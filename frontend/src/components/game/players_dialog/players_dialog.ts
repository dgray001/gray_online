import { DwgDialogBox } from '../../dialog_box/dialog_box';
import type { LobbyUser } from '../../lobby/data_models';
import type { GamePlayer } from '../data_models';

import html from './players_dialog.html';

import './players_dialog.scss';
import './players_dialog_player/players_dialog_player';

/** Input data for a players dialog box */
interface PlayersData {
  room_id: number;
  players: GamePlayer[];
  lobby_players: Map<number, LobbyUser>;
}

export class DwgPlayersDialog extends DwgDialogBox<PlayersData> {
  private close_button!: HTMLButtonElement;
  private players_container!: HTMLDivElement;

  private data!: PlayersData;

  constructor() {
    super();
    this.configureElements('close_button', 'players_container');
  }

  override getHTML(): string {
    return html;
  }

  getData(): PlayersData {
    return this.data;
  }

  setData(data: PlayersData, parsed?: boolean) {
    this.data = data;
    if (!parsed && !this.fully_parsed) {
      return;
    }
    for (const player of this.data.players) {
      const lobby_player = this.data.lobby_players.get(player.client_id);
      const el = document.createElement('dwg-players-dialog-player');
      el.id = `player-${player.player_id}`;
      el.classList.add('player');
      el.setData(player, lobby_player, this.data.room_id);
      this.players_container.appendChild(el);
    }
    this.close_button.addEventListener('click', () => {
      this.closeDialog();
    });
  }
}

customElements.define('dwg-players-dialog', DwgPlayersDialog);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-players-dialog': DwgPlayersDialog;
  }
}
