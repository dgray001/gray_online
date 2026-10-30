import { DwgElement } from '../../../../dwg_element';
import type { DwgDialogBox } from '../../../../dialog_box/dialog_box';
import type { FiddlesticksSettings } from '../../../../../scripts/settings_store';
import {
  defaultFiddlesticksSettings,
  FIDDLESTICKS_CARD_SIZES,
  getSettings,
  updateSettings,
} from '../../../../../scripts/settings_store';

import html from './fiddlesticks_settings.html';

import './fiddlesticks_settings.scss';

export class DwgFiddlesticksSettings extends DwgElement {
  private animation_speed!: HTMLInputElement;
  private animation_speed_value!: HTMLOutputElement;
  private card_size!: HTMLInputElement;
  private card_size_value!: HTMLOutputElement;
  private trick_result_pause!: HTMLInputElement;
  private trick_result_pause_value!: HTMLOutputElement;
  private highlight_playable_cards!: HTMLInputElement;
  private show_ai_comparisons!: HTMLInputElement;
  private restore_defaults_button!: HTMLButtonElement;
  private cancel_changes_button!: HTMLButtonElement;
  private save_changes_button!: HTMLButtonElement;
  private settings: FiddlesticksSettings = { ...getSettings().fiddlesticks };
  constructor() {
    super();
    this.html_string = html;
    this.configureElements(
      'animation_speed',
      'animation_speed_value',
      'card_size',
      'card_size_value',
      'trick_result_pause',
      'trick_result_pause_value',
      'highlight_playable_cards',
      'show_ai_comparisons',
      'restore_defaults_button',
      'cancel_changes_button',
      'save_changes_button'
    );
  }
  protected override parsedCallback(): void {
    this.render();
    this.animation_speed.addEventListener('input', (): void => {
      this.settings.animation_speed = Number(this.animation_speed.value);
      this.render();
    });
    this.card_size.addEventListener('input', (): void => {
      this.settings.card_size = FIDDLESTICKS_CARD_SIZES[Number(this.card_size.value)];
      this.render();
    });
    this.trick_result_pause.addEventListener('input', (): void => {
      this.settings.trick_result_pause = Number(this.trick_result_pause.value);
      this.render();
    });
    this.highlight_playable_cards.addEventListener('change', (): void => {
      this.settings.highlight_playable_cards = this.highlight_playable_cards.checked;
    });
    this.show_ai_comparisons.addEventListener('change', (): void => {
      this.settings.show_ai_comparisons = this.show_ai_comparisons.checked;
    });
    this.restore_defaults_button.addEventListener('click', (): void => {
      this.settings = defaultFiddlesticksSettings();
      this.render();
    });
    this.cancel_changes_button.addEventListener('click', (): void => this.closeDialog());
    this.save_changes_button.addEventListener('click', (): void => {
      updateSettings({ fiddlesticks: { ...this.settings } });
      this.closeDialog();
    });
  }

  private render(): void {
    this.animation_speed.value = this.settings.animation_speed.toString();
    this.animation_speed_value.textContent = `${this.settings.animation_speed}×`;
    const size_index = FIDDLESTICKS_CARD_SIZES.indexOf(this.settings.card_size);
    const size_label = ['Small', 'Normal', 'Large'][size_index];
    this.card_size.value = size_index.toString();
    this.card_size.setAttribute('aria-valuetext', size_label);
    this.card_size_value.textContent = size_label;
    this.trick_result_pause.value = this.settings.trick_result_pause.toString();
    this.trick_result_pause_value.textContent = `${this.settings.trick_result_pause}s`;
    this.highlight_playable_cards.checked = this.settings.highlight_playable_cards;
    this.show_ai_comparisons.checked = this.settings.show_ai_comparisons;
  }

  private closeDialog(): void {
    const dialog = this.closest<DwgDialogBox<unknown>>('.dwg-dialog-box');
    if (dialog) {
      dialog.closeDialog();
    } else {
      this.remove();
    }
  }
}

customElements.define('dwg-fiddlesticks-settings', DwgFiddlesticksSettings);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-fiddlesticks-settings': DwgFiddlesticksSettings;
  }
}
