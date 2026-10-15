import type { BoardTransformData } from '../../canvas_board/canvas_board';
import { drawLine, drawRect } from '../../canvas_util';
import type { Point2D } from '../../objects2d';
import { configDraw } from '../canvas_component';
import { DwgSlider } from './slider';
import type { RectSliderConfig } from './slider_config';

const DEFAULT_TRACK_HEIGHT = 4;
const DEFAULT_THUMB_WIDTH = 14;
const DEFAULT_MARKER_SIZE = 8;

export class DwgRectSlider extends DwgSlider {
  private rect_config: RectSliderConfig;

  constructor(config: RectSliderConfig) {
    super(config.slider_config);
    this.rect_config = config;
  }

  xi(): number {
    return this.rect_config.p.x;
  }

  xf(): number {
    return this.rect_config.p.x + this.rect_config.w;
  }

  yi(): number {
    return this.rect_config.p.y;
  }

  yf(): number {
    return this.rect_config.p.y + this.rect_config.h;
  }

  w(): number {
    return this.rect_config.w;
  }

  h(): number {
    return this.rect_config.h;
  }

  setPosition(p: Point2D): void {
    this.rect_config.p = { ...p };
  }

  setSize(w: number, h: number): void {
    this.rect_config.w = w;
    this.rect_config.h = h;
  }

  protected override mouseOver(canvas: Point2D, screen: Point2D): boolean {
    const pt = this.rect_config.draw_config.fixed_position ? screen : canvas;
    const p = this.rect_config.p;
    return pt.x >= p.x && pt.x <= p.x + this.rect_config.w && pt.y >= p.y && pt.y <= p.y + this.rect_config.h;
  }

  protected override fractionFromPoint(canvas: Point2D, screen: Point2D): number {
    const pt = this.rect_config.draw_config.fixed_position ? screen : canvas;
    const thumb_w = this.rect_config.thumb_width ?? DEFAULT_THUMB_WIDTH;
    const track_inset = thumb_w / 2;
    const track_w = Math.max(1, this.rect_config.w - 2 * track_inset);
    return (pt.x - (this.rect_config.p.x + track_inset)) / track_w;
  }

  protected override changed(value: number): void {
    this.rect_config.onChange?.(value);
  }

  protected override _draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, _dt: number): void {
    const p = this.rect_config.p;
    const w = this.rect_config.w;
    const h = this.rect_config.h;
    const thumb_w = this.rect_config.thumb_width ?? DEFAULT_THUMB_WIDTH;
    const thumb_h = this.rect_config.thumb_height ?? Math.max(12, h - 4);
    const track_h = this.rect_config.track_height ?? DEFAULT_TRACK_HEIGHT;
    const track_inset = thumb_w / 2;
    const track_w = Math.max(1, w - 2 * track_inset);
    const track_x = p.x + track_inset;
    const track_y = p.y + (h - track_h) / 2;
    const fraction = this.valueFraction();
    const thumb_cx = track_x + fraction * track_w;

    const track_config = this.rect_config.track_draw_config ?? {
      fill_style: 'rgb(50, 50, 50)',
      stroke_style: 'rgb(80, 80, 80)',
      stroke_width: 1,
    };
    configDraw(ctx, transform, track_config, false, false, () => {
      drawRect(ctx, { x: track_x, y: track_y }, track_w, track_h, track_h / 2);
    });

    if (fraction > 0) {
      const active_config = {
        fill_style: this.rect_config.draw_config.click_stroke_style ?? 'rgb(60, 140, 240)',
        stroke_width: 0,
        fixed_position: this.rect_config.draw_config.fixed_position,
      };
      configDraw(ctx, transform, active_config, false, false, () => {
        drawRect(ctx, { x: track_x, y: track_y }, fraction * track_w, track_h, track_h / 2);
      });
    }

    if (this.showMarkers()) {
      const fixed = this.fixedValues();
      if (fixed && fixed.length > 1) {
        const min = this.minValue();
        const max = this.maxValue();
        const span = max - min;
        const marker_size = this.rect_config.marker_size ?? DEFAULT_MARKER_SIZE;
        const marker_y = p.y + h / 2;
        const marker_config = {
          fill_style: 'transparent',
          stroke_style: this.rect_config.marker_color ?? 'rgba(255, 255, 255, 0.4)',
          stroke_width: 2,
          fixed_position: this.rect_config.draw_config.fixed_position,
        };
        configDraw(ctx, transform, marker_config, false, false, () => {
          for (const val of fixed) {
            const mx = track_x + (span > 0 ? (val - min) / span : 0) * track_w;
            drawLine(ctx, { x: mx, y: marker_y - marker_size / 2 }, { x: mx, y: marker_y + marker_size / 2 });
          }
        });
      }
    }

    const thumb_x = thumb_cx - thumb_w / 2;
    const thumb_y = p.y + (h - thumb_h) / 2;
    const thumb_r = this.rect_config.r ?? 3;
    configDraw(ctx, transform, this.rect_config.draw_config, this.isHovering(), this.isClicking(), () => {
      drawRect(ctx, { x: thumb_x, y: thumb_y }, thumb_w, thumb_h, thumb_r);
    });
  }
}
