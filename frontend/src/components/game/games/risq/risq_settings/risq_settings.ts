import { DwgElement } from '../../../../dwg_element';
import { apiGet } from '../../../../../scripts/api';
import { getSettings, updateSettings } from '../../../../../scripts/settings_store';
import { enumKeys } from '../../../../../scripts/util';
import type { HotkeyCombo, RisqHotkeyBindings } from '../risq_hotkeys';
import {
  RISQ_HOTKEY_ACTION_LABELS,
  RisqHotkeyAction,
  comboFromKeyboardEvent,
  comboFromMouseEvent,
  comboToDisplayString,
  comboToString,
  isUnbound,
} from '../risq_hotkeys';
import { RisqProducibleKind } from '../risq_data';

import html from './risq_settings.html';

import './risq_settings.scss';

interface ProducibleEntry {
  kind: RisqProducibleKind;
  id: number;
  display_name: string;
  unlocks_mercenaries?: { id: number; display_name: string }[];
}

interface BuildingTreeEntry {
  building_id: number;
  display_name: string;
  produces: ProducibleEntry[];
}

type IdBindingGroup = 'create_unit' | 'research_tech' | 'build_building' | 'cycle_building' | 'hire_mercenary';

type BindingSlot = { group: 'actions'; key: RisqHotkeyAction } | { group: IdBindingGroup; key: number };

interface HotkeyEntry {
  label: string;
  slot: BindingSlot;
}

type RisqSettingsTab = 'animations' | 'hotkeys';

export class DwgRisqSettings extends DwgElement {
  private tab_button_animations!: HTMLButtonElement;
  private tab_button_hotkeys!: HTMLButtonElement;
  private tab_content_animations!: HTMLDivElement;
  private tab_content_hotkeys!: HTMLDivElement;
  private status!: HTMLDivElement;
  private rows!: HTMLDivElement;
  private bindings: RisqHotkeyBindings = getSettings().risq_hotkeys;
  private entries: HotkeyEntry[] = [];
  private listener_cleanup?: () => void;

  constructor() {
    super();
    this.html_string = html;
    this.configureElements(
      'tab_button_animations',
      'tab_button_hotkeys',
      'tab_content_animations',
      'tab_content_hotkeys',
      'status',
      'rows'
    );
  }

  protected override async parsedCallback(): Promise<void> {
    this.tab_button_animations.addEventListener('click', () => this.showTab('animations'));
    this.tab_button_hotkeys.addEventListener('click', () => this.showTab('hotkeys'));
    for (const key of enumKeys(RisqHotkeyAction)) {
      const action = RisqHotkeyAction[key];
      this.entries.push({ label: RISQ_HOTKEY_ACTION_LABELS[action], slot: { group: 'actions', key: action } });
    }
    this.renderAll();
    const response = await apiGet<BuildingTreeEntry[]>('risq/tech-tree');
    this.status.classList.add('hide');
    if (response.success) {
      this.addBuildingEntries(response.result);
      this.renderAll();
    }
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    this.listener_cleanup?.();
  }

  private showTab(tab: RisqSettingsTab) {
    this.tab_button_animations.classList.toggle('active', tab === 'animations');
    this.tab_button_hotkeys.classList.toggle('active', tab === 'hotkeys');
    this.tab_content_animations.classList.toggle('active', tab === 'animations');
    this.tab_content_hotkeys.classList.toggle('active', tab === 'hotkeys');
  }

  private addBuildingEntries(buildings: BuildingTreeEntry[]) {
    for (const building of buildings) {
      if (building.produces.length === 0) {
        continue;
      }
      this.entries.push({
        label: `Select Building: ${building.display_name}`,
        slot: { group: 'cycle_building', key: building.building_id },
      });
      for (const producible of building.produces) {
        const [group, verb] = this.producibleGroup(producible.kind);
        this.entries.push({ label: `${verb}: ${producible.display_name}`, slot: { group, key: producible.id } });
        for (const mercenary of producible.unlocks_mercenaries ?? []) {
          this.entries.push({
            label: `Hire: ${mercenary.display_name}`,
            slot: { group: 'hire_mercenary', key: mercenary.id },
          });
        }
      }
    }
  }

  private producibleGroup(kind: RisqProducibleKind): [IdBindingGroup, string] {
    switch (kind) {
      case RisqProducibleKind.UNIT:
        return ['create_unit', 'Create'];
      case RisqProducibleKind.TECH:
        return ['research_tech', 'Research'];
      default:
        return ['build_building', 'Build'];
    }
  }

  private slotCombo(slot: BindingSlot): HotkeyCombo {
    return slot.group === 'actions' ? this.bindings.actions[slot.key] : (this.bindings[slot.group][slot.key] ?? {});
  }

  private assign(slot: BindingSlot, combo: HotkeyCombo) {
    if (slot.group === 'actions') {
      this.bindings.actions[slot.key] = combo;
    } else {
      this.bindings[slot.group][slot.key] = combo;
    }
  }

  private setSlotCombo(slot: BindingSlot, combo: HotkeyCombo) {
    if (!isUnbound(combo)) {
      const target = comboToString(combo);
      for (const entry of this.entries) {
        if (comboToString(this.slotCombo(entry.slot)) === target) {
          this.assign(entry.slot, {});
        }
      }
    }
    this.assign(slot, combo);
    updateSettings({ risq_hotkeys: this.bindings });
  }

  private renderAll() {
    this.rows.replaceChildren(...this.entries.map((entry) => this.buildRow(entry)));
  }

  private buildRow(entry: HotkeyEntry): HTMLDivElement {
    const row = document.createElement('div');
    row.classList.add('hotkey-row');
    const label_el = document.createElement('span');
    label_el.classList.add('hotkey-label');
    label_el.innerText = entry.label;
    const value_button = document.createElement('button');
    value_button.classList.add('hotkey-value');
    value_button.innerText = this.valueText(entry.slot);
    value_button.addEventListener('click', () => this.beginListening(value_button, entry.slot));
    const clear_button = document.createElement('button');
    clear_button.classList.add('hotkey-clear');
    clear_button.innerText = 'x';
    clear_button.addEventListener('click', () => {
      this.setSlotCombo(entry.slot, {});
      this.renderAll();
    });
    row.append(label_el, value_button, clear_button);
    return row;
  }

  private valueText(slot: BindingSlot): string {
    const combo = this.slotCombo(slot);
    return isUnbound(combo) ? 'Unbound' : comboToDisplayString(combo);
  }

  private beginListening(button: HTMLButtonElement, slot: BindingSlot) {
    this.listener_cleanup?.();
    button.innerText = 'Press a key...';
    const on_keydown = (e: KeyboardEvent) => {
      e.preventDefault();
      if (e.key === 'Escape') {
        this.stopListening(button, slot);
        return;
      }
      if (/^[0-9]$/.test(e.key) || ['Control', 'Shift', 'Alt'].includes(e.key)) {
        return; // digits are reserved for control groups; a bare modifier isn't a complete combo
      }
      this.setSlotCombo(slot, comboFromKeyboardEvent(e));
      this.renderAll();
      this.listener_cleanup?.();
      this.listener_cleanup = undefined;
    };
    const on_mousedown = (e: MouseEvent) => {
      if (e.button < 3) {
        return; // 0/1/2 are reserved for game interaction
      }
      e.preventDefault();
      this.setSlotCombo(slot, comboFromMouseEvent(e));
      this.renderAll();
      this.listener_cleanup?.();
      this.listener_cleanup = undefined;
    };
    window.addEventListener('keydown', on_keydown, true);
    window.addEventListener('mousedown', on_mousedown, true);
    this.listener_cleanup = () => {
      window.removeEventListener('keydown', on_keydown, true);
      window.removeEventListener('mousedown', on_mousedown, true);
    };
  }

  private stopListening(button: HTMLButtonElement, slot: BindingSlot) {
    this.listener_cleanup?.();
    this.listener_cleanup = undefined;
    button.innerText = this.valueText(slot);
  }
}

customElements.define('dwg-risq-settings', DwgRisqSettings);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-risq-settings': DwgRisqSettings;
  }
}
