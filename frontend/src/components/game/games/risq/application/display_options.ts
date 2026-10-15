import type { RisqDisplaySettings } from '../../../../../scripts/settings_store';

export declare interface DisplayOption<T> {
  value: T;
  label: string;
}

export declare interface DisplaySpec<T> {
  label: string;
  control: 'slider' | 'checkbox';
  options: DisplayOption<T>[];
}

const px = (...values: number[]): DisplayOption<number>[] => values.map((value) => ({ value, label: `${value}px` }));
const percent = (...values: number[]): DisplayOption<number>[] =>
  values.map((value) => ({ value, label: `${Math.round(100 * value)}%` }));
const slider = <T>(label: string, options: DisplayOption<T>[]): DisplaySpec<T> => ({
  label,
  control: 'slider',
  options,
});

const ENABLED_OPTIONS: DisplayOption<boolean>[] = [
  { value: false, label: 'Disabled' },
  { value: true, label: 'Enabled' },
];

/** Every display setting's dialog row and its valid values, in row order; the store validates against these too */
export const RISQ_DISPLAY_SPECS: { [K in keyof RisqDisplaySettings]: DisplaySpec<RisqDisplaySettings[K]> } = {
  unit_circle_size: slider('Unit circle size', px(28, 36, 44, 52, 60)),
  healthbars: slider('Healthbars', [
    { value: 'never', label: 'Never' },
    { value: 'selected_only', label: 'Selected only' },
    { value: 'damaged_only', label: 'Damaged only' },
    { value: 'always', label: 'Always' },
  ]),
  secondary_bars: slider('Secondary progress / resources', [
    { value: 'never', label: 'Never' },
    { value: 'selected_only', label: 'Selected only' },
    { value: 'active_only', label: 'Active only' },
    { value: 'always', label: 'Always' },
  ]),
  healthbar_size: slider('Healthbar size', px(2, 4, 6, 8)),
  hud_size: slider('HUD size', percent(0.85, 1, 1.15)),
  auto_open_turn_report: slider('Auto-open turn report', ENABLED_OPTIONS),
  order_visibility: slider('Order lines & arrows', [
    { value: 'none', label: 'None' },
    { value: 'selected_only', label: 'Selected only' },
    { value: 'all', label: 'All' },
  ]),
  order_size: slider('Order & gather flag size', percent(0.75, 1, 1.25, 1.5)),
  territory_border_width: slider('Territory border width', percent(0.7, 1, 1.4)),
  selection_outline_thickness: slider('Selection outline thickness', px(1, 2, 3, 4)),
  idle_indicator_size: slider('Idle indicator size', [{ value: 0, label: 'Off' }, ...px(8, 12, 16, 20)]),
  idle_indicator_pulse: { label: 'Pulse idle indicators', control: 'checkbox', options: ENABLED_OPTIONS },
};

export const RISQ_DISPLAY_KEYS = Object.keys(RISQ_DISPLAY_SPECS) as (keyof RisqDisplaySettings)[];
