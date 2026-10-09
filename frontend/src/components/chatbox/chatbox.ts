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
  private chat_container!: HTMLDivElement;
  private chat_input!: HTMLInputElement;
  private send_chat!: HTMLButtonElement;
  private new_messages_button!: HTMLButtonElement;
  private new_messages_number!: HTMLSpanElement;

  convert_emoticons = true;

  constructor() {
    super();
    this.html_string = html;
    this.configureElements('chat_container', 'chat_input', 'send_chat', 'new_messages_button', 'new_messages_number');
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

  adjustScroll(): void {
    this.chat_container.scrollTop = this.chat_container.scrollHeight - this.chat_container.offsetHeight;
    this.scrolledToBottom();
  }

  scrolledToBottom(): void {
    this.last_new_messages_button_count = 0;
    this.new_messages_button.classList.remove('show');
  }

  private static adjust_scroll_limit = 5;
  private last_new_messages_button_timer?: number;
  private last_new_messages_button_count = 0;

  addChat(message: ChatMessage, you_sent = false): void {
    const scrolled_up = this.scrolledUp();
    const sender = !!message.sender && message.sender !== SERVER_CHAT_NAME ? `${message.sender}: ` : '';
    const new_element = document.createElement('div');
    if (message.sender === SERVER_CHAT_NAME || message.color === 'gray') {
      new_element.classList.add('color-gray');
    }
    const sender_el = document.createElement('b');
    sender_el.innerText = sender;
    new_element.appendChild(sender_el);
    new_element.appendChild(document.createTextNode(message.message));
    this.chat_container.appendChild(new_element);
    new_element.classList.add('new-message');
    window.setTimeout(() => new_element.classList.remove('new-message'), 1500);
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
    this.last_new_messages_button_timer = window.setTimeout(() => {
      this.new_messages_button.classList.remove('new-message');
      this.classList.remove('new-message');
      this.last_new_messages_button_timer = undefined;
    }, 1500);
  }

  sendChat(): void {
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

  setPlaceholder(placeholder: string): void {
    this.chat_input.placeholder = placeholder;
  }

  focus(): void {
    this.chat_input.focus();
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    if (this.last_new_messages_button_timer) {
      clearTimeout(this.last_new_messages_button_timer);
      this.last_new_messages_button_timer = undefined;
    }
  }

  clear(): void {
    if (this.last_new_messages_button_timer) {
      clearTimeout(this.last_new_messages_button_timer);
      this.last_new_messages_button_timer = undefined;
    }
    this.chat_input.value = '';
    this.chat_container.innerHTML = '';
    this.classList.remove('new-message');
    this.new_messages_button.classList.remove('new-message');
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
