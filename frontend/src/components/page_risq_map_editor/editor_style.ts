import type { DrawConfig } from '../game/util/canvas_components/canvas_component';

export const HEADER_H = 36;
export const CONTROL_H = 24;
export const FONT = '12px "Fira Sans", sans-serif';
export const INK = 'rgb(59, 36, 19)';
export const PANEL_BACKGROUND = 'rgb(222, 184, 135)';
export const TEXT_CONFIG = { font: FONT, fill_style: INK };
export const CONTROL_DRAW: DrawConfig = {
  fill_style: 'rgb(241, 226, 196)',
  hover_fill_style: 'rgb(255, 243, 210)',
  click_fill_style: 'rgb(230, 210, 170)',
  stroke_style: INK,
  stroke_width: 0.8,
  fixed_position: true,
};
