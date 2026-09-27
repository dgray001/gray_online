import { enumKeys } from '../../../../scripts/util';

/** A configurable hotkey: a letter key or an extra mouse button (index 3+), plus modifiers. Never a digit — those are reserved for control groups. */
export declare interface HotkeyCombo {
  key?: string; // lowercase letter, e.g. 'q'
  mouse_button?: number; // extra mouse button index (3+; 0/1/2 are reserved for game interaction)
  ctrl?: boolean;
  shift?: boolean;
  alt?: boolean;
}

const UNBOUND: HotkeyCombo = {};

export function isUnbound(combo: HotkeyCombo): boolean {
  return !combo.key && combo.mouse_button === undefined;
}

/** Canonical string key for a combo, used for reverse lookup; empty string for unbound */
export function comboToString(combo: HotkeyCombo): string {
  if (isUnbound(combo)) {
    return '';
  }
  const parts: string[] = [];
  if (combo.ctrl) {
    parts.push('ctrl');
  }
  if (combo.shift) {
    parts.push('shift');
  }
  if (combo.alt) {
    parts.push('alt');
  }
  parts.push(combo.key !== undefined ? combo.key : `mouse${combo.mouse_button}`);
  return parts.join('+');
}

export function comboFromKeyboardEvent(e: KeyboardEvent): HotkeyCombo {
  return {
    key: e.key.toLowerCase(),
    ctrl: e.ctrlKey,
    shift: e.shiftKey,
    alt: e.altKey,
  };
}

export function hotkeyDisplayString(combo: HotkeyCombo | undefined): string {
  return !combo || isUnbound(combo) ? 'Unset' : comboToDisplayString(combo);
}

export function comboToDisplayString(combo: HotkeyCombo): string {
  const is_letter = combo.key !== undefined && /^[a-z]$/.test(combo.key);
  const parts: string[] = [];
  if (combo.ctrl) {
    parts.push('Ctrl');
  }
  if (combo.shift && !is_letter) {
    parts.push('Shift');
  }
  if (combo.alt) {
    parts.push('Alt');
  }
  if (combo.key === undefined) {
    parts.push(`Mouse ${combo.mouse_button}`);
  } else if (is_letter) {
    parts.push(combo.shift ? combo.key.toUpperCase() : combo.key);
  } else {
    parts.push(combo.key.toUpperCase());
  }
  return parts.join('+');
}

export function comboFromMouseEvent(e: MouseEvent): HotkeyCombo {
  return {
    mouse_button: e.button,
    ctrl: e.ctrlKey,
    shift: e.shiftKey,
    alt: e.altKey,
  };
}

/** Fixed number of control groups (digits 0-9); not user-configurable, not part of RisqHotkeyBindings */
export const CONTROL_GROUP_COUNT = 10;

export enum RisqHotkeyAction {
  CYCLE_VIEW_MODE,
  VIEW_MODE_ALL,
  VIEW_MODE_RESOURCE,
  VIEW_MODE_MILITARY,
  VIEW_MODE_OWNERSHIP,
  VIEW_MODE_REGION,
  SUMMARY_REPORT,
  TECH_TREE,
  NEXT_IDLE,
  SUBMIT_ORDERS,
  MOVE,
  ATTACK,
  STOP,
  DELETE,
  GARRISON,
  UNGARRISON,
  GATHER,
  REPAIR,
  RENEW,
  STANCE_AGGRESSIVE,
  STANCE_DEFENSIVE,
  STANCE_STAND_GROUND,
  STANCE_PASSIVE,
  TOGGLE_INTERRUPT_CURRENT,
  TOGGLE_ATTACK_BACK,
  BUILDING_ATTACK,
  BUILDING_GATHER_POINT,
  BUILDING_DELETE,
}

export const RISQ_HOTKEY_ACTION_LABELS: Record<RisqHotkeyAction, string> = {
  [RisqHotkeyAction.CYCLE_VIEW_MODE]: 'Cycle View Mode',
  [RisqHotkeyAction.VIEW_MODE_ALL]: 'View Mode: All',
  [RisqHotkeyAction.VIEW_MODE_RESOURCE]: 'View Mode: Resource',
  [RisqHotkeyAction.VIEW_MODE_MILITARY]: 'View Mode: Military',
  [RisqHotkeyAction.VIEW_MODE_OWNERSHIP]: 'View Mode: Ownership',
  [RisqHotkeyAction.VIEW_MODE_REGION]: 'View Mode: Region',
  [RisqHotkeyAction.SUMMARY_REPORT]: 'Summary Report',
  [RisqHotkeyAction.TECH_TREE]: 'Tech Tree',
  [RisqHotkeyAction.NEXT_IDLE]: 'Next Idle Unit',
  [RisqHotkeyAction.SUBMIT_ORDERS]: 'Submit Orders',
  [RisqHotkeyAction.MOVE]: 'Move',
  [RisqHotkeyAction.ATTACK]: 'Attack',
  [RisqHotkeyAction.STOP]: 'Stop',
  [RisqHotkeyAction.DELETE]: 'Delete',
  [RisqHotkeyAction.GARRISON]: 'Garrison',
  [RisqHotkeyAction.UNGARRISON]: 'Ungarrison',
  [RisqHotkeyAction.GATHER]: 'Gather',
  [RisqHotkeyAction.REPAIR]: 'Repair',
  [RisqHotkeyAction.RENEW]: 'Renew',
  [RisqHotkeyAction.STANCE_AGGRESSIVE]: 'Stance: Aggressive',
  [RisqHotkeyAction.STANCE_DEFENSIVE]: 'Stance: Defensive',
  [RisqHotkeyAction.STANCE_STAND_GROUND]: 'Stance: Stand Ground',
  [RisqHotkeyAction.STANCE_PASSIVE]: 'Stance: Passive',
  [RisqHotkeyAction.TOGGLE_INTERRUPT_CURRENT]: 'Toggle Interrupt Current',
  [RisqHotkeyAction.TOGGLE_ATTACK_BACK]: 'Toggle Attack Back',
  [RisqHotkeyAction.BUILDING_ATTACK]: 'Building: Attack',
  [RisqHotkeyAction.BUILDING_GATHER_POINT]: 'Building: Gather Point',
  [RisqHotkeyAction.BUILDING_DELETE]: 'Building: Delete',
};

/** Fixed actions plus config-driven, id-keyed ones (units/techs/buildings are per-game-config, not enumerable statically) */
export declare interface RisqHotkeyBindings {
  actions: Record<RisqHotkeyAction, HotkeyCombo>;
  create_unit: Record<number, HotkeyCombo>;
  research_tech: Record<number, HotkeyCombo>;
  build_building: Record<number, HotkeyCombo>;
  cycle_building: Record<number, HotkeyCombo>; // cycle selection through all of a building type
}

export function defaultRisqHotkeyBindings(): RisqHotkeyBindings {
  const actions = {} as Record<RisqHotkeyAction, HotkeyCombo>;
  for (const key of enumKeys(RisqHotkeyAction)) {
    actions[RisqHotkeyAction[key]] = { ...UNBOUND };
  }
  return {
    actions,
    create_unit: {},
    research_tech: {},
    build_building: {},
    cycle_building: {},
  };
}

export type RisqHotkeyLookupEntry =
  | { kind: 'action'; action: RisqHotkeyAction }
  | { kind: 'create_unit'; id: number }
  | { kind: 'research_tech'; id: number }
  | { kind: 'build_building'; id: number }
  | { kind: 'cycle_building'; id: number };

/** Builds a combo-string -> entry lookup for dispatch; unbound combos are omitted */
export function buildHotkeyLookup(bindings: RisqHotkeyBindings): Map<string, RisqHotkeyLookupEntry> {
  const lookup = new Map<string, RisqHotkeyLookupEntry>();
  const add = (combo: HotkeyCombo, entry: RisqHotkeyLookupEntry) => {
    if (isUnbound(combo)) {
      return;
    }
    lookup.set(comboToString(combo), entry);
  };
  for (const key of enumKeys(RisqHotkeyAction)) {
    const action = RisqHotkeyAction[key];
    add(bindings.actions[action], { kind: 'action', action });
  }
  for (const [id, combo] of Object.entries(bindings.create_unit)) {
    add(combo, { kind: 'create_unit', id: Number(id) });
  }
  for (const [id, combo] of Object.entries(bindings.research_tech)) {
    add(combo, { kind: 'research_tech', id: Number(id) });
  }
  for (const [id, combo] of Object.entries(bindings.build_building)) {
    add(combo, { kind: 'build_building', id: Number(id) });
  }
  for (const [id, combo] of Object.entries(bindings.cycle_building)) {
    add(combo, { kind: 'cycle_building', id: Number(id) });
  }
  return lookup;
}
