import { DwgElement } from '../dwg_element';

import html from './goldset.html';

import './goldset.scss';
import type { GoldItem, QueueItem } from './types';
import { request } from './api';
import { actionName, element } from './dom';
import { renderState } from './render_state';
import { renderChoices } from './render_choices';
import { renderAnalysis } from './render_analysis';
import { renderHistory } from './render_history';

export class DwgGoldset extends DwgElement {
  private queue!: HTMLDivElement;
  private state!: HTMLDivElement;
  private choices!: HTMLDivElement;
  private analysis!: HTMLElement;
  private history!: HTMLDivElement;
  private notes!: HTMLTextAreaElement;
  private confidence!: HTMLSelectElement;
  private save!: HTMLButtonElement;
  private filter!: HTMLSelectElement;
  private counts!: HTMLElement;
  private selection!: HTMLParagraphElement;
  private status!: HTMLParagraphElement;
  private answer_form!: HTMLFormElement;
  private generate_form!: HTMLFormElement;
  private skip!: HTMLButtonElement;
  private generate!: HTMLButtonElement;
  private items: QueueItem[] = [];
  private current: GoldItem | null = null;
  private selected: number | null = null;
  private acceptable: number[] = [];
  private equivalents!: HTMLInputElement;
  private busy = false;

  constructor() {
    super();
    this.html_string = html;
    this.configureElements(
      'queue',
      'state',
      'choices',
      'equivalents',
      'analysis',
      'history',
      'notes',
      'confidence',
      'save',
      'filter',
      'counts',
      'selection',
      'status',
      'answer_form',
      'generate_form',
      'skip',
      'generate'
    );
  }

  protected override parsedCallback(): void {
    this.answer_form.onsubmit = (event: SubmitEvent): void => {
      event.preventDefault();
      this.perform((): Promise<void> => this.saveLabel());
    };
    this.generate_form.onsubmit = (event: SubmitEvent): void => {
      event.preventDefault();
      this.perform((): Promise<void> => this.addItems());
    };
    this.filter.onchange = (): void => this.perform((): Promise<void> => this.reloadQueue());
    this.skip.onclick = (): void => this.perform((): Promise<void> => this.skipItem());
    this.equivalents.onchange = (): void => {
      if (!this.equivalents.checked) {
        this.acceptable = this.selected === null ? [] : [this.selected];
      }
      this.renderSelection();
    };
    this.perform((): Promise<void> => this.reloadQueue());
  }

  private perform(action: () => Promise<void>): void {
    if (this.busy) {
      return;
    }
    this.busy = true;
    this.updateControls();
    this.status.textContent = 'Working…';
    void action()
      .catch((error: unknown): void => {
        this.status.textContent = error instanceof Error ? error.message : String(error);
      })
      .finally((): void => {
        this.busy = false;
        this.updateControls();
      });
  }

  private updateControls(): void {
    this.equivalents.disabled = this.busy || !this.current;
    this.save.disabled = this.busy || this.selected === null;
    this.skip.disabled = this.busy || !this.current;
    this.generate.disabled = this.busy;
    this.filter.disabled = this.busy;
    this.notes.disabled = this.busy;
    this.confidence.disabled = this.busy;
  }

  private filtered(): QueueItem[] {
    return this.items.filter(
      (item: QueueItem): boolean =>
        this.filter.value === 'all' || (this.filter.value === 'labelled' ? item.labelled : !item.labelled)
    );
  }

  private renderQueue(): void {
    this.queue.replaceChildren();
    const done = this.items.filter((item: QueueItem): boolean => item.labelled).length;
    this.counts.textContent = `${this.items.length - done} open · ${done} labelled`;
    for (const item of this.filtered()) {
      const button = element(
        'button',
        `${item.kind === 'bet' ? 'Bid' : 'Play'} · ${item.players} players · ${item.hand_size} cards · P${item.seat + 1}`
      );
      button.type = 'button';
      button.classList.toggle('active', item.id === this.current?.id);
      button.onclick = (): void => this.perform((): Promise<void> => this.openItem(item.id));
      this.queue.append(button);
    }
  }

  private async reloadQueue(preferred: string | null = null): Promise<void> {
    const result = await request<{ items: QueueItem[] }>('/api/items');
    this.items = result.items;
    this.renderQueue();
    const visible = this.filtered();
    const target = visible.find((item: QueueItem): boolean => item.id === preferred)?.id ?? visible[0]?.id;
    if (target) {
      if (target !== this.current?.id) {
        await this.openItem(target);
      } else {
        this.status.textContent = 'Queue updated.';
      }
      return;
    }
    this.current = null;
    this.selected = null;
    this.acceptable = [];
    this.equivalents.checked = false;
    this.answer_form.hidden = true;
    this.state.replaceChildren(element('h2', 'No situations in this view. Add more or change the filter.'));
    this.choices.replaceChildren();
    this.history.replaceChildren();
    this.analysis.replaceChildren();
    this.selection.textContent = '';
    this.status.textContent = 'Queue updated.';
  }

  private async openItem(id: string): Promise<void> {
    const item = await request<GoldItem>(`/api/items/${id}`);
    this.current = item;
    this.selected = item.label?.action ?? null;
    this.acceptable = item.label?.acceptable_actions?.slice() ?? (this.selected === null ? [] : [this.selected]);
    this.equivalents.checked = this.acceptable.length > 1;
    this.answer_form.hidden = false;
    renderState(this.state, item.state);
    renderChoices(this.choices, item, (action: number): void => this.choose(action));
    renderAnalysis(this.analysis, item);
    renderHistory(this.history, item.state);
    this.notes.value = item.label?.notes ?? '';
    this.confidence.value = String(item.label?.confidence ?? 1);
    this.renderQueue();
    this.renderSelection();
    this.status.textContent = item.label
      ? `Saved ${new Date(item.label.saved_at).toLocaleString()}. Editing creates a revision.`
      : 'Choose the action you would take.';
  }

  private choose(action: number | null): void {
    if (action === null) {
      this.acceptable = [];
    } else if (!this.equivalents.checked) {
      this.acceptable = [action];
    } else if (this.acceptable.includes(action)) {
      this.acceptable = this.acceptable.filter((value: number): boolean => value !== action);
    } else {
      this.acceptable.push(action);
    }
    if (this.selected === null || !this.acceptable.includes(this.selected)) {
      this.selected = this.acceptable[0] ?? null;
    }
    this.renderSelection();
  }

  private renderSelection(): void {
    this.selection.textContent =
      this.acceptable.length === 0 || !this.current
        ? 'Choose an action to label this situation.'
        : `Acceptable: ${this.acceptable.map((action: number): string => actionName(this.current!.state, action)).join(' OR ')}`;
    for (const button of this.choices.querySelectorAll<HTMLButtonElement>('button[data-action]')) {
      const selected = this.acceptable.includes(Number(button.dataset.action));
      button.classList.toggle('selected', selected);
      button.setAttribute('aria-pressed', String(selected));
    }
    this.updateControls();
  }

  private async saveLabel(): Promise<void> {
    if (!this.current || this.selected === null) {
      return;
    }
    const visible = this.filtered();
    const index = visible.findIndex((item: QueueItem): boolean => item.id === this.current?.id);
    const next = visible[index + 1]?.id ?? visible[0]?.id ?? null;
    this.current = await request<GoldItem>(`/api/items/${this.current.id}/label`, {
      action: this.selected,
      acceptable_actions: this.acceptable,
      notes: this.notes.value,
      confidence: Number(this.confidence.value),
    });
    await this.reloadQueue(next);
    this.status.textContent = 'Gold label saved.';
  }

  private async skipItem(): Promise<void> {
    const visible = this.filtered();
    const index = visible.findIndex((item: QueueItem): boolean => item.id === this.current?.id);
    const next = visible[(index + 1) % visible.length];
    if (next) {
      await this.openItem(next.id);
    }
    this.status.textContent = 'Skipped; the situation remains open.';
  }

  private async addItems(): Promise<void> {
    const data = new FormData(this.generate_form);
    const number = (name: string): number => Number(data.get(name));
    const result = await request<{ added: number }>('/api/items/generate', {
      count: number('count'),
      players: number('players'),
      min_hand: number('min_hand'),
      max_hand: number('max_hand'),
      kind: String(data.get('kind')),
      round_points: number('round_points'),
      trick_points: number('trick_points'),
      seed: number('seed'),
    });
    const seed = this.generate_form.elements.namedItem('seed') as HTMLInputElement;
    seed.value = String(number('seed') + 1);
    await this.reloadQueue(this.current?.id ?? null);
    this.status.textContent = `Added ${result.added} situations.`;
  }
}

customElements.define('dwg-goldset', DwgGoldset);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-goldset': DwgGoldset;
  }
}
