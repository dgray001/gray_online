import {DwgElement} from '../dwg_element';
import {clickButton} from '../../scripts/util';

import html from './chatbox.html';
import './chatbox.scss';

/** Data describing a chat message */
export declare interface ChatMessage {
  message: string;
  sender?: string;
  color?: string;
}

export class DwgChatbox extends DwgElement {
  chat_container: HTMLDivElement;
  chat_input: HTMLInputElement;
  send_chat: HTMLButtonElement;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('chat_container');
    this.configureElement('chat_input');
    this.configureElement('send_chat');
  }

  protected override parsedCallback(): void {
    clickButton(this.send_chat, () => {
      this.sendChat();
    });
    this.chat_input.addEventListener('keyup', (e) => {
      if (e.key === 'Enter') {
        this.sendChat();
      }
    });
  }

  static adjust_scroll_limit = 5;
  addChat(message: ChatMessage) {
    const scroll_height = this.chat_container.scrollHeight - this.chat_container.offsetHeight;
    const adjust_scroll = (scroll_height - this.chat_container.scrollTop) < DwgChatbox.adjust_scroll_limit;
    const open_tag = message.sender === 'server' || message.color === 'gray' ?
      '<div class="color-gray">' : '<div>'
    const sender = !!message.sender && message.sender !== 'server' ?
      `${message.sender}: ` : '';
    this.chat_container.innerHTML += `${open_tag}<b>${sender}</b>${message.message}</div>`;
    if (adjust_scroll) {
      this.chat_container.scrollTop = this.chat_container.scrollHeight - this.chat_container.offsetHeight;
    } else {
      // TODO: let you know new messages below
    }
  }

  sendChat() {
    if (this.inputEmpty()) {
      return;
    }
    const chat_event = new CustomEvent('chat_sent', {'detail': this.getInput()});
    this.dispatchEvent(chat_event);
  }

  setPlaceholder(placeholder: string) {
    this.chat_input.placeholder = placeholder;
  }

  clear() {
    this.chat_input.value = '';
    this.chat_container.innerHTML = '';
  }

  private inputEmpty(): boolean {
    return !this.chat_input.value;
  }

  private getInput(): ChatMessage {
    const message: ChatMessage = {
      message: this.chat_input.value,
    };
    this.chat_input.value = '';
    return message;
  }
}

customElements.define('dwg-chatbox', DwgChatbox);
