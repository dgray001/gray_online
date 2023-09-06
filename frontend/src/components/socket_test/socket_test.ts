import {DwgElement} from '../dwg_element';

import html from './socket_test.html';
import './socket_test.scss';

export class DwgSocketTest extends DwgElement {
  example: HTMLDivElement;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('example', 'example');
  }

  protected override parsedCallback(): void {
    const socket = new WebSocket('ws://127.0.0.1:6807/api/ws');
    console.log(socket);
    socket.addEventListener('error', (e) => {
      console.log(e);
    });
    socket.addEventListener('open', (e) => {
      console.log('socket opened', e);
      socket.addEventListener('message', (e) => {
        console.log(e);
      });
    });
  }
}

customElements.define('dwg-socket-test', DwgSocketTest);
