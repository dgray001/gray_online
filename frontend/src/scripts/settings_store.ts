import type { RisqHotkeyBindings } from '../components/game/games/risq/application/input/hotkeys';
import { defaultRisqHotkeyBindings } from '../components/game/games/risq/application/input/hotkeys';

const STORAGE_KEY = 'gray_online_settings';

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

// Fills newly added actions with their defaults while preserving saved bindings.
function mergeRisqHotkeyBindings(parsed?: Partial<RisqHotkeyBindings>): RisqHotkeyBindings {
  const base = defaultRisqHotkeyBindings();
  if (!parsed) {
    return base;
  }
  return {
    actions: { ...base.actions, ...parsed.actions },
    create_unit: { ...parsed.create_unit },
    research_tech: { ...parsed.research_tech },
    build_building: { ...parsed.build_building },
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
