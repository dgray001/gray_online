import { copyToClipboard } from '../../../../scripts/util';
import { DwgElement } from '../../../dwg_element';
import type { LobbyUser } from '../../../lobby/data_models';
import type { GamePlayer } from '../../data_models';

import html from './players_dialog_player.html';

import './players_dialog_player.scss';

export class DwgPlayersDialogPlayer extends DwgElement {
  private ping_text: HTMLDivElement;
  private ping_image: HTMLImageElement;
  private nickname: HTMLDivElement;
  private rejoin_link: HTMLButtonElement;
  private rejoin_link_text: HTMLButtonElement;

  private rejoin_url: string;
  private player: GamePlayer;
  private room_id: number;
  private lobby_player: LobbyUser | undefined;
  private copy_timeout: NodeJS.Timeout;

  constructor() {
    super();
    this.html_string = html;
    this.configureElement('ping_text');
    this.configureElement('ping_image');
    this.configureElement('nickname');
    this.configureElement('rejoin_link');
    this.configureElement('rejoin_link_text');
  }

  protected override parsedCallback(): void {
    if (!this.player) {
      console.error('Must set player before attaching to dom');
      return;
    }
    if (!this.lobby_player || !this.player.connected) {
      this.ping_image.src = '/images/icons/ping_disconnected24.png';
      this.ping_text.innerText = 'Disconnected';
    } else {
      this.ping_text.innerText = `${Math.round(this.lobby_player.ping)} ms`;
      if (this.lobby_player.ping < 100) {
        this.ping_image.src = '/images/icons/ping_good24.png';
      } else if (this.lobby_player.ping < 300) {
        this.ping_image.src = '/images/icons/ping_ok24.png';
      } else if (this.lobby_player.ping < 1000) {
        this.ping_image.src = '/images/icons/ping_bad24.png';
      } else {
        this.ping_image.src = '/images/icons/ping_empty24.png';
      }
    }
    this.nickname.innerText = this.player.nickname;
    this.rejoin_url = `${window.location.origin}${window.location.pathname}?room_id=${this.room_id}&client_id=${this.player.client_id}`;
    this.rejoin_link.addEventListener('click', async () => {
      const copied = await copyToClipboard(this.rejoin_url);
      if (copied) {
        this.rejoin_link_text.innerText = 'Copied!';
        if (!!this.copy_timeout) {
          clearTimeout(this.copy_timeout);
        }
        this.copy_timeout = setTimeout(() => {
          this.rejoin_link_text.innerText = 'Copy Again';
        }, 1500);
      } else {
        this.rejoin_link_text.innerText = 'Failed to Copy';
        if (!!this.copy_timeout) {
          clearTimeout(this.copy_timeout);
        }
        this.copy_timeout = setTimeout(() => {
          this.rejoin_link_text.innerText = this.rejoin_url;
        }, 1500);
      }
    });
  }

  setData(player: GamePlayer, lobby_player: LobbyUser, room_id: number) {
    this.player = player;
    this.lobby_player = lobby_player;
    this.room_id = room_id;
  }
}

customElements.define('dwg-players-dialog-player', DwgPlayersDialogPlayer);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-players-dialog-player': DwgPlayersDialogPlayer;
  }
}
