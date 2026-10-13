import type { HotkeyCombo, RisqHotkeyBindings } from '../components/game/games/risq/application/input/hotkeys';
import { defaultRisqHotkeyBindings, RisqHotkeyAction } from '../components/game/games/risq/application/input/hotkeys';

const STORAGE_KEY = 'gray_online_settings';

const LEGACY_HOTKEY_ACTIONS = (
  'CYCLE_VIEW_MODE VIEW_MODE_ALL VIEW_MODE_RESOURCE VIEW_MODE_MILITARY VIEW_MODE_OWNERSHIP VIEW_MODE_REGION ' +
  'SUMMARY_REPORT TECH_TREE NEXT_IDLE SUBMIT_ORDERS MOVE ATTACK STOP DELETE GARRISON UNGARRISON GATHER REPAIR RENEW ' +
  'STANCE_AGGRESSIVE STANCE_DEFENSIVE STANCE_STAND_GROUND STANCE_PASSIVE TOGGLE_INTERRUPT_CURRENT TOGGLE_ATTACK_BACK ' +
  'BUILDING_ATTACK BUILDING_GATHER_POINT BUILDING_DELETE BUILDING_CLEAR_GATHER_POINT GO_TO_SELECTION'
).split(' ');

export declare interface AppSettings {
  last_nickname: string;
  client_id: number;
  client_id_time: number;
  risq_hotkeys: RisqHotkeyBindings;
}

const DEFAULT_SETTINGS: AppSettings = {
  last_nickname: '',
  client_id: 0,
  client_id_time: 0,
  risq_hotkeys: defaultRisqHotkeyBindings(),
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

function load(): AppSettings {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    const parsed = raw ? JSON.parse(raw) : undefined;
    return { ...DEFAULT_SETTINGS, ...parsed, risq_hotkeys: mergeRisqHotkeyBindings(parsed?.risq_hotkeys) };
  } catch (_e) {
    return { ...DEFAULT_SETTINGS };
  }
}

let settings = load();

export function getSettings(): AppSettings {
  return settings;
}

export function updateSettings(patch: Partial<AppSettings>): void {
  settings = { ...settings, ...patch };
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
  } catch (_e) {}
}
