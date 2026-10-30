export const WALL_MERLONS = 20;
export const WALL_THICKNESS_SCALE = 0.5;
export const HEIGHT_SCALE = 0.75;

const PARAPET_PERIODS = 0.5;
const DECK_PERIODS = 1.3;
const MERLON_WIDTH_PERIODS = 0.5;
const MERLON_HEIGHT_PERIODS = 1.0;
const CRENEL_HEIGHT_PERIODS = 0.42;
const WALL_HEIGHT_RATIO = 0.13;
const WALL_SCALE = 0.8;
export const FLAG_SCALE = 0.6;
export const TOWER_SCALE = 0.9;

export const WALL_DEPTH_RATIO = ((PARAPET_PERIODS + DECK_PERIODS) * WALL_SCALE) / WALL_MERLONS;

export declare interface WallMetrics {
  hex_r: number;
  period: number;
  unit: number;
  wall_h: number;
  parapet_depth: number;
  deck_depth: number;
  merlon_w: number;
  merlon_h: number;
  merlon_depth: number;
  crenel_h: number;
}

export function wallMetrics(hex_r: number): WallMetrics {
  const period = hex_r / WALL_MERLONS;
  const unit = period * WALL_SCALE;
  return {
    hex_r,
    period,
    unit,
    wall_h: WALL_HEIGHT_RATIO * hex_r * WALL_SCALE * HEIGHT_SCALE,
    parapet_depth: PARAPET_PERIODS * unit,
    deck_depth: DECK_PERIODS * unit,
    merlon_w: MERLON_WIDTH_PERIODS * period,
    merlon_h: MERLON_HEIGHT_PERIODS * unit * HEIGHT_SCALE,
    merlon_depth: PARAPET_PERIODS * unit,
    crenel_h: CRENEL_HEIGHT_PERIODS * unit * HEIGHT_SCALE,
  };
}
