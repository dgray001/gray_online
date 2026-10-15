import type { DrawConfig } from '../canvas_component';
import type { Point2D } from '../../objects2d';

/** Configuration for a base canvas slider */
export declare interface SliderConfig {
  /** Minimum slider value; defaults to 0 */
  min?: number;
  /** Maximum slider value; defaults to 1 */
  max?: number;
  /** Step size for snapping; defaults to 0 (continuous) */
  step?: number;
  /** Initial value */
  value?: number;
  /** Fixed discrete values if the slider snaps to predetermined points */
  fixed_values?: number[];
  /** Whether to render tick markers on the track */
  show_markers?: boolean;
  /** If true, starts disabled */
  start_disabled?: boolean;
}

/** Configuration for a rectangular canvas slider */
export declare interface RectSliderConfig {
  slider_config: SliderConfig;
  p: Point2D;
  w: number;
  h: number;
  draw_config: DrawConfig;
  track_draw_config?: DrawConfig;
  track_height?: number;
  thumb_width?: number;
  thumb_height?: number;
  r?: number;
  marker_color?: string;
  marker_size?: number;
  onChange?: (value: number) => void;
}
