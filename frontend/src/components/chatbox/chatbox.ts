import {DwgElement} from '../dwg_element';
import {clickButton} from '../../scripts/util';

import html from './chatbox.html';
import './chatbox.scss';

/** Data describing a chat message */
export declare interface ChatMessage {
  sender?: string;
  message: string;
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
  }

  addChat(message: ChatMessage) {
    this.chat_container.innerHTML += message.message;
  }

  sendChat() {
    if (this.inputEmpty()) {
      return;
    }
    const chat_event = new CustomEvent('chat-sent', {'detail': this.getInput()});
    this.dispatchEvent(chat_event);
    
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
