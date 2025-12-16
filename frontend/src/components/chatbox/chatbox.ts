import { DwgElement } from '../dwg_element';
import { clickButton } from '../../scripts/util';
import { emoticons } from '../../data/emoji_data';

import html from './chatbox.html';
import './chatbox.scss';

/** Data describing a chat message */
export declare interface ChatMessage {
  message: string;
  sender?: string;
  color?: string;
}

/** Sender name reserved for server chats */
export const SERVER_CHAT_NAME = '!!server!!';

export class DwgChatbox extends DwgElement {
  chat_container: HTMLDivElement;
  chat_input: HTMLInputElement;
  send_chat: HTMLButtonElement;
  new_messages_button: HTMLButtonElement;
  new_messages_number: HTMLSpanElement;

  convert_emoticons = true;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('chat_container');
    this.configureElement('chat_input');
    this.configureElement('send_chat');
    this.configureElement('new_messages_button');
    this.configureElement('new_messages_number');
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
    this.new_messages_button.addEventListener('click', () => {
      this.adjustScroll();
    });
    this.chat_container.addEventListener('scroll', () => {
      if (!this.scrolledUp()) {
        this.scrolledToBottom();
      }
    });
  }

  scrolledUp(): boolean {
    const scroll_height = this.chat_container.scrollHeight - this.chat_container.offsetHeight;
    return scroll_height - this.chat_container.scrollTop > DwgChatbox.adjust_scroll_limit;
  }

  adjustScroll() {
    this.chat_container.scrollTop = this.chat_container.scrollHeight - this.chat_container.offsetHeight;
    this.scrolledToBottom();
  }

  scrolledToBottom() {
    this.last_new_messages_button_count = 0;
    this.new_messages_button.classList.remove('show');
  }

  static adjust_scroll_limit = 5;
  last_new_messages_button_timer: number;
  last_new_messages_button_count = 0;
  new_chat_elements: HTMLDivElement[] = [];
  addChat(message: ChatMessage, you_sent = false) {
    const scrolled_up = this.scrolledUp();
    const sender = !!message.sender && message.sender !== SERVER_CHAT_NAME ? `${message.sender}: ` : '';
    const new_element = document.createElement('div');
    if (message.sender === SERVER_CHAT_NAME || message.color === 'gray') {
      new_element.classList.add('color-gray');
    }
    new_element.innerHTML = `<b>${sender}</b>${message.message}`;
    this.chat_container.appendChild(new_element);
    new_element.classList.add('new-message');
    this.new_chat_elements.push(new_element);
    this.classList.add('new-message');
    if (this.last_new_messages_button_timer) {
      clearTimeout(this.last_new_messages_button_timer);
    }
    if (you_sent || !scrolled_up || (this.classList.contains('transparent-fade') && !this.classList.contains('show'))) {
      this.adjustScroll();
    } else {
      this.new_messages_button.classList.add('show');
      this.new_messages_button.classList.add('new-message');
      this.last_new_messages_button_count++;
      this.new_messages_number.innerText = this.last_new_messages_button_count.toString();
    }
    setTimeout(() => {
      this.new_messages_button.classList.remove('new-message');
      this.new_chat_elements.forEach((element) => {
        element.classList.remove('new-message');
      });
      this.classList.remove('new-message');
    }, 1500);
  }

  sendChat() {
    if (this.inputEmpty()) {
      return;
    }
    const chat_input = this.getInput();
    if (this.convert_emoticons) {
      for (const [emoticon, emoji] of emoticons) {
        chat_input.message = chat_input.message.replace(emoticon, emoji);
      }
    }
    const chat_event = new CustomEvent('chat_sent', { detail: chat_input });
    this.dispatchEvent(chat_event);
  }

  setPlaceholder(placeholder: string) {
    this.chat_input.placeholder = placeholder;
  }

  focus() {
    this.chat_input.focus();
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

declare global {
  interface HTMLElementTagNameMap {
    'dwg-chatbox': DwgChatbox;
  }
}
