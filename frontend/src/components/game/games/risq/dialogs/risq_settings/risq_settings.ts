import { DwgElement } from '../../../../../dwg_element';
import type { DwgDialogBox } from '../../../../../dialog_box/dialog_box';
import { apiGet } from '../../../../../../scripts/api';
import type { RisqDisplaySettings } from '../../../../../../scripts/settings_store';
import { defaultRisqDisplaySettings, getSettings, updateSettings } from '../../../../../../scripts/settings_store';
import { enumKeys } from '../../../../../../scripts/util';
import type {
  HotkeyCombo,
  HotkeySelection,
  RisqHotkeyBindings,
  RisqHotkeyLookupEntry,
} from '../../application/input/hotkeys';
import {
  RISQ_HOTKEY_ACTION_LABELS,
  RISQ_HOTKEY_SCOPE_LABELS,
  RisqHotkeyAction,
  RisqHotkeyScope,
  UNIT_HOTKEY_SELECTIONS,
  actionScope,
  comboFromKeyboardEvent,
  comboFromMouseEvent,
  comboToDisplayString,
  comboToString,
  defaultRisqHotkeyBindings,
  hotkeysConflict,
  isUnbound,
} from '../../application/input/hotkeys';
import { RisqProducibleKind } from '../../model/types';
import type { DisplayOption, DisplaySpec } from '../../application/display_options';
import { RISQ_DISPLAY_KEYS, RISQ_DISPLAY_SPECS } from '../../application/display_options';

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
  buildable: boolean;
  produces: ProducibleEntry[];
}

type IdBindingGroup = 'create_unit' | 'research_tech' | 'build_building' | 'cycle_building' | 'hire_mercenary';

type BindingSlot = { group: 'actions'; key: RisqHotkeyAction } | { group: IdBindingGroup; key: number };

interface HotkeyEntry {
  section: string;
  subsection?: string;
  label: string;
  slot: BindingSlot;
}

const GLOBAL_SECTION = RISQ_HOTKEY_SCOPE_LABELS[RisqHotkeyScope.GLOBAL];

type RisqSettingsTab = 'display' | 'hotkeys';

export class DwgRisqSettings extends DwgElement {
  private tab_button_display!: HTMLButtonElement;
  private tab_button_hotkeys!: HTMLButtonElement;
  private tab_content_display!: HTMLDivElement;
  private tab_content_hotkeys!: HTMLDivElement;
  private status!: HTMLDivElement;
  private rows!: HTMLDivElement;
  private display_rows!: HTMLDivElement;
  private restore_defaults_button!: HTMLButtonElement;
  private cancel_changes_button!: HTMLButtonElement;
  private save_changes_button!: HTMLButtonElement;
  private bindings: RisqHotkeyBindings = structuredClone(getSettings().risq_hotkeys);
  private display_settings: RisqDisplaySettings = structuredClone(getSettings().risq_display);
  private entries: HotkeyEntry[] = [];
  private selections: HotkeySelection[] = [...UNIT_HOTKEY_SELECTIONS, { kind: 'building', unit_ids: [], tech_ids: [] }];
  private listener_cleanup?: () => void;

  constructor() {
    super();
    this.html_string = html;
    this.configureElements(
      'tab_button_display',
      'tab_button_hotkeys',
      'tab_content_display',
      'tab_content_hotkeys',
      'display_rows',
      'restore_defaults_button',
      'cancel_changes_button',
      'save_changes_button',
      'status',
      'rows'
    );
  }

  protected override async parsedCallback(): Promise<void> {
    this.entries = [];
    this.tab_button_display.addEventListener('click', () => this.showTab('display'));
    this.tab_button_hotkeys.addEventListener('click', () => this.showTab('hotkeys'));
    this.restore_defaults_button.addEventListener('click', () => this.restoreDefaults());
    this.cancel_changes_button.addEventListener('click', () => this.cancelChanges());
    this.save_changes_button.addEventListener('click', () => this.saveChanges());
    for (const key of enumKeys(RisqHotkeyAction)) {
      const action = RisqHotkeyAction[key];
      this.entries.push({
        section: RISQ_HOTKEY_SCOPE_LABELS[actionScope(action)],
        subsection: action === RisqHotkeyAction.GO_TO_SELECTION ? 'Selection' : undefined,
        label: RISQ_HOTKEY_ACTION_LABELS[action],
        slot: { group: 'actions', key: action },
      });
    }
    this.renderDisplay();
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

  private showTab(tab: RisqSettingsTab): void {
    this.tab_button_display.classList.toggle('active', tab === 'display');
    this.tab_button_hotkeys.classList.toggle('active', tab === 'hotkeys');
    this.tab_content_display.classList.toggle('active', tab === 'display');
    this.tab_content_hotkeys.classList.toggle('active', tab === 'hotkeys');
    if (tab === 'display') {
      this.renderDisplay();
    } else {
      this.renderAll();
    }
  }

  private addBuildingEntries(buildings: BuildingTreeEntry[]): void {
    for (const building of buildings) {
      if (building.buildable) {
        this.entries.push({
          section: RISQ_HOTKEY_SCOPE_LABELS[RisqHotkeyScope.VILLAGERS],
          label: `Build: ${building.display_name}`,
          slot: { group: 'build_building', key: building.building_id },
        });
      }
      if (building.produces.length === 0) {
        continue;
      }
      const ids = (kind: RisqProducibleKind): number[] =>
        building.produces.filter((p) => p.kind === kind).map((p) => p.id);
      this.selections.push({
        kind: 'building',
        unit_ids: ids(RisqProducibleKind.UNIT),
        tech_ids: ids(RisqProducibleKind.TECH),
      });
      this.entries.push({
        section: GLOBAL_SECTION,
        label: `Select Building: ${building.display_name}`,
        subsection: 'Selection',
        slot: { group: 'cycle_building', key: building.building_id },
      });
      for (const producible of building.produces) {
        const [group, verb] = this.producibleGroup(producible.kind);
        const section =
          group === 'build_building' ? RISQ_HOTKEY_SCOPE_LABELS[RisqHotkeyScope.VILLAGERS] : building.display_name;
        this.entries.push({
          section,
          label: `${verb}: ${producible.display_name}`,
          slot: { group, key: producible.id },
        });
        for (const mercenary of producible.unlocks_mercenaries ?? []) {
          this.entries.push({
            section: GLOBAL_SECTION,
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

  private assign(slot: BindingSlot, combo: HotkeyCombo): void {
    if (slot.group === 'actions') {
      this.bindings.actions[slot.key] = combo;
    } else {
      this.bindings[slot.group][slot.key] = combo;
    }
  }

  private slotEntry(slot: BindingSlot): RisqHotkeyLookupEntry {
    return slot.group === 'actions' ? { kind: 'action', action: slot.key } : { kind: slot.group, id: slot.key };
  }

  private setSlotCombo(slot: BindingSlot, combo: HotkeyCombo): void {
    if (!isUnbound(combo)) {
      const target = comboToString(combo);
      const new_entry = this.slotEntry(slot);
      for (const entry of this.entries) {
        const same_combo = comboToString(this.slotCombo(entry.slot)) === target;
        if (same_combo && hotkeysConflict(this.slotEntry(entry.slot), new_entry, this.selections)) {
          this.assign(entry.slot, {});
        }
      }
    }
    this.assign(slot, combo);
  }

  private renderAll(): void {
    this.cancelListening();
    const sections = new Map<string, HotkeyEntry[]>();
    for (const entry of this.entries) {
      sections.set(entry.section, [...(sections.get(entry.section) ?? []), entry]);
    }
    this.rows.replaceChildren(
      ...[...sections.entries()].flatMap(([section, entries]) => {
        const header = document.createElement('div');
        header.classList.add('hotkey-section-header');
        header.innerText = section;
        return [header, ...this.buildSectionRows(entries)];
      })
    );
  }

  private buildSectionRows(entries: HotkeyEntry[]): HTMLDivElement[] {
    const subsections = new Map<string, HotkeyEntry[]>();
    const rows = entries.filter((entry) => !entry.subsection).map((entry) => this.buildRow(entry));
    for (const entry of entries) {
      if (entry.subsection) {
        subsections.set(entry.subsection, [...(subsections.get(entry.subsection) ?? []), entry]);
      }
    }
    for (const [subsection, subsection_entries] of subsections) {
      const header = document.createElement('div');
      header.classList.add('hotkey-subsection-header');
      header.innerText = subsection;
      rows.push(header, ...subsection_entries.map((entry) => this.buildRow(entry)));
    }
    return rows;
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

  private beginListening(button: HTMLButtonElement, slot: BindingSlot): void {
    this.listener_cleanup?.();
    button.innerText = 'Press a key...';
    const on_keydown = (e: KeyboardEvent): void => {
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
    };
    const on_mousedown = (e: MouseEvent): void => {
      if (e.button < 3) {
        return; // 0/1/2 are reserved for game interaction
      }
      e.preventDefault();
      this.setSlotCombo(slot, comboFromMouseEvent(e));
      this.renderAll();
    };
    window.addEventListener('keydown', on_keydown, true);
    window.addEventListener('mousedown', on_mousedown, true);
    this.listener_cleanup = () => {
      window.removeEventListener('keydown', on_keydown, true);
      window.removeEventListener('mousedown', on_mousedown, true);
    };
  }

  private cancelListening(): void {
    this.listener_cleanup?.();
    this.listener_cleanup = undefined;
  }

  private updateDisplaySetting(key: keyof RisqDisplaySettings, value: unknown): void {
    this.display_settings = { ...this.display_settings, [key]: value };
  }

  /** A native range input stepping through the option indices, with a datalist for tick marks */
  private createSlider<T>(
    label: string,
    options: DisplayOption<T>[],
    index: number,
    on_select: (opt: DisplayOption<T>) => void
  ): [HTMLInputElement, HTMLDataListElement] {
    const slider = document.createElement('input');
    slider.type = 'range';
    slider.min = '0';
    slider.max = (options.length - 1).toString();
    slider.step = '1';
    slider.value = index.toString();
    slider.classList.add('display-slider');
    slider.addEventListener('input', () => on_select(options[Number(slider.value)]));
    const ticks = document.createElement('datalist');
    ticks.id = `ticks-${label.toLowerCase().replace(/[^a-z0-9]+/g, '-')}`;
    ticks.append(
      ...options.map((_, i): HTMLOptionElement => {
        const tick = document.createElement('option');
        tick.value = i.toString();
        return tick;
      })
    );
    slider.setAttribute('list', ticks.id);
    return [slider, ticks];
  }

  private buildSliderRow<T>(
    label: string,
    options: DisplayOption<T>[],
    current_value: T,
    on_change: (value: T) => void
  ): HTMLDivElement {
    const row = document.createElement('div');
    row.classList.add('display-row');
    const label_el = document.createElement('span');
    label_el.classList.add('display-label');
    label_el.innerText = label;
    const value_el = document.createElement('span');
    value_el.classList.add('display-value');
    const index = Math.max(
      0,
      options.findIndex((opt) => opt.value === current_value)
    );
    value_el.innerText = options[index]?.label ?? '';
    const [slider, ticks] = this.createSlider(label, options, index, (opt) => {
      value_el.innerText = opt.label;
      on_change(opt.value);
    });
    row.append(label_el, slider, value_el, ticks);
    return row;
  }

  private buildCheckboxRow(
    spec: DisplaySpec<unknown>,
    current: unknown,
    on_change: (value: unknown) => void
  ): HTMLDivElement {
    const row = document.createElement('div');
    row.classList.add('display-row');
    const label_el = document.createElement('span');
    label_el.classList.add('display-label');
    label_el.innerText = spec.label;
    const input = document.createElement('input');
    input.type = 'checkbox';
    input.classList.add('display-checkbox');
    input.checked = current === true;
    const val_el = document.createElement('span');
    val_el.classList.add('display-value');
    const label_for = (checked: boolean): string =>
      spec.options.find((option) => option.value === checked)?.label ?? '';
    val_el.innerText = label_for(input.checked);
    input.addEventListener('change', () => {
      val_el.innerText = label_for(input.checked);
      on_change(input.checked);
    });
    row.append(label_el, input, val_el);
    return row;
  }

  private displayRow(key: keyof RisqDisplaySettings): HTMLDivElement {
    const spec: DisplaySpec<unknown> = RISQ_DISPLAY_SPECS[key];
    const current = this.display_settings[key];
    const on_change = (value: unknown): void => this.updateDisplaySetting(key, value);
    return spec.control === 'checkbox'
      ? this.buildCheckboxRow(spec, current, on_change)
      : this.buildSliderRow(spec.label, spec.options, current, on_change);
  }

  private renderDisplay(): void {
    this.display_rows.replaceChildren(...RISQ_DISPLAY_KEYS.map((key) => this.displayRow(key)));
  }

  private restoreDefaults(): void {
    this.cancelListening();
    this.display_settings = defaultRisqDisplaySettings();
    this.bindings = defaultRisqHotkeyBindings();
    this.renderDisplay();
    this.renderAll();
  }

  private saveChanges(): void {
    this.cancelListening();
    updateSettings({
      risq_display: this.display_settings,
      risq_hotkeys: this.bindings,
    });
    this.closeDialog();
  }

  private cancelChanges(): void {
    this.closeDialog();
  }

  private closeDialog(): void {
    this.cancelListening();
    const dialog = this.closest<DwgDialogBox<unknown>>('.dwg-dialog-box');
    if (dialog) {
      dialog.closeDialog();
    } else {
      this.remove();
    }
  }

  private stopListening(button: HTMLButtonElement, slot: BindingSlot): void {
    this.cancelListening();
    button.innerText = this.valueText(slot);
  }
}

customElements.define('dwg-risq-settings', DwgRisqSettings);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-risq-settings': DwgRisqSettings;
  }
}
