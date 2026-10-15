import type { HotkeyCombo, RisqHotkeyBindings } from '../components/game/games/risq/application/input/hotkeys';
import { RISQ_DISPLAY_KEYS, RISQ_DISPLAY_SPECS } from '../components/game/games/risq/application/display_options';
import { defaultRisqHotkeyBindings, RisqHotkeyAction } from '../components/game/games/risq/application/input/hotkeys';

const STORAGE_KEY = 'gray_online_settings';

const LEGACY_HOTKEY_ACTIONS = (
  'CYCLE_VIEW_MODE VIEW_MODE_ALL VIEW_MODE_RESOURCE VIEW_MODE_MILITARY VIEW_MODE_OWNERSHIP VIEW_MODE_REGION ' +
  'SUMMARY_REPORT TECH_TREE NEXT_IDLE SUBMIT_ORDERS MOVE ATTACK STOP DELETE GARRISON UNGARRISON GATHER REPAIR RENEW ' +
  'STANCE_AGGRESSIVE STANCE_DEFENSIVE STANCE_STAND_GROUND STANCE_PASSIVE TOGGLE_INTERRUPT_CURRENT TOGGLE_ATTACK_BACK ' +
  'BUILDING_ATTACK BUILDING_GATHER_POINT BUILDING_DELETE BUILDING_CLEAR_GATHER_POINT GO_TO_SELECTION'
).split(' ');

export declare interface RisqDisplaySettings {
  unit_circle_size: number;
  healthbars: 'always' | 'damaged_only' | 'selected_only' | 'never';
  secondary_bars: 'always' | 'active_only' | 'selected_only' | 'never';
  healthbar_size: number;
  hud_size: number;
  auto_open_turn_report: boolean;
  order_visibility: 'none' | 'selected_only' | 'all';
  order_size: number;
  territory_border_width: number;
  selection_outline_thickness: number;
  idle_indicator_size: number;
  idle_indicator_pulse: boolean;
}

export function defaultRisqDisplaySettings(): RisqDisplaySettings {
  return {
    unit_circle_size: 44,
    healthbars: 'damaged_only',
    secondary_bars: 'active_only',
    healthbar_size: 4,
    hud_size: 1.0,
    auto_open_turn_report: true,
    order_visibility: 'all',
    order_size: 1.0,
    territory_border_width: 1.0,
    selection_outline_thickness: 2,
    idle_indicator_size: 0,
    idle_indicator_pulse: false,
  };
}

export interface FiddlesticksSettings {
  animation_speed: number;
  card_size: number;
  trick_result_pause: number;
  highlight_playable_cards: boolean;
}

export const FIDDLESTICKS_ANIMATION_SPEEDS: readonly number[] = [0, 0.5, 1, 1.5];
export const FIDDLESTICKS_CARD_SIZES: readonly number[] = [0.8, 1, 1.2];
export const FIDDLESTICKS_TRICK_PAUSES: readonly number[] = [0, 0.5, 1, 1.5, 2];

export function defaultFiddlesticksSettings(): FiddlesticksSettings {
  return { animation_speed: 1, card_size: 1, trick_result_pause: 0.5, highlight_playable_cards: true };
}

export declare interface AppSettings {
  last_nickname: string;
  client_id: number;
  client_id_time: number;
  risq_hotkeys: RisqHotkeyBindings;
  risq_display: RisqDisplaySettings;
  fiddlesticks: FiddlesticksSettings;
}

const DEFAULT_SETTINGS: AppSettings = {
  last_nickname: '',
  client_id: 0,
  client_id_time: 0,
  risq_hotkeys: defaultRisqHotkeyBindings(),
  risq_display: defaultRisqDisplaySettings(),
  fiddlesticks: defaultFiddlesticksSettings(),
};

function migrateRisqHotkeyActions(
  saved: Partial<Record<string, HotkeyCombo>> = {}
): Partial<RisqHotkeyBindings['actions']> {
  const actions: Partial<RisqHotkeyBindings['actions']> = {};
  for (const [key, combo] of Object.entries(saved)) {
    const action = /^\d+$/.test(key) ? LEGACY_HOTKEY_ACTIONS[Number(key)] : key;
    if (action in RisqHotkeyAction && combo !== undefined) {
      actions[action as RisqHotkeyAction] = saved[action] ?? combo;
    }
  }
  return actions;
}

// Fills newly added actions with their defaults while preserving saved bindings.
function mergeRisqHotkeyBindings(parsed?: Partial<RisqHotkeyBindings>): RisqHotkeyBindings {
  const base = defaultRisqHotkeyBindings();
  if (!parsed) {
    return base;
  }
  return {
    actions: { ...base.actions, ...migrateRisqHotkeyActions(parsed.actions) },
    create_unit: { ...base.create_unit, ...parsed.create_unit },
    research_tech: { ...base.research_tech, ...parsed.research_tech },
    build_building: { ...base.build_building, ...parsed.build_building },
    cycle_building: { ...parsed.cycle_building },
    hire_mercenary: { ...parsed.hire_mercenary },
  };
}

/** Keeps each saved display value only if the dialog offers it, so stale or corrupted values fall back to the default */
function sanitizeRisqDisplay(saved: Partial<Record<keyof RisqDisplaySettings, unknown>> = {}): RisqDisplaySettings {
  const display = defaultRisqDisplaySettings();
  for (const key of RISQ_DISPLAY_KEYS) {
    if (RISQ_DISPLAY_SPECS[key].options.some((option) => option.value === saved[key])) {
      Object.assign(display, { [key]: saved[key] });
    }
  }
  return display;
}

function sanitizeFiddlesticksSettings(
  saved?: Partial<Record<keyof FiddlesticksSettings, unknown>>
): FiddlesticksSettings {
  const settings = defaultFiddlesticksSettings();
  const numeric_options = [
    ['animation_speed', FIDDLESTICKS_ANIMATION_SPEEDS],
    ['card_size', FIDDLESTICKS_CARD_SIZES],
    ['trick_result_pause', FIDDLESTICKS_TRICK_PAUSES],
  ] as const;
  for (const [key, options] of numeric_options) {
    const value = saved?.[key];
    if (typeof value === 'number' && options.includes(value)) {
      settings[key] = value;
    }
  }
  if (typeof saved?.highlight_playable_cards === 'boolean') {
    settings.highlight_playable_cards = saved.highlight_playable_cards;
  }
  return settings;
}

function load(): AppSettings {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    const parsed = raw ? JSON.parse(raw) : undefined;
    return {
      ...DEFAULT_SETTINGS,
      ...parsed,
      risq_hotkeys: mergeRisqHotkeyBindings(parsed?.risq_hotkeys),
      risq_display: sanitizeRisqDisplay(parsed?.risq_display),
      fiddlesticks: sanitizeFiddlesticksSettings(parsed?.fiddlesticks),
    };
  } catch (_e) {
    return { ...DEFAULT_SETTINGS };
  }
}

let settings = load();

export function getSettings(): AppSettings {
  return settings;
}

const change_listeners = new Set<() => void>();

/** Calls the listener after every settings update; returns the unsubscribe function */
export function onSettingsChanged(listener: () => void): () => void {
  change_listeners.add(listener);
  return (): void => void change_listeners.delete(listener);
}

export function updateSettings(patch: Partial<AppSettings>): void {
  settings = { ...settings, ...patch };
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
  } catch (_e) {}
  change_listeners.forEach((listener) => listener());
}
