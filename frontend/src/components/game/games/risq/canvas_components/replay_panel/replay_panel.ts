import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { DwgRectSlider } from '../../../../util/canvas_components/slider/rect_slider';
import { drawRect } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import { RISQ_MESSAGE_WARNING_COLOR } from '../message_queue';
import type { ReplayGlyph } from './replay_control_button';
import { REPLAY_CONTROL_SIZE, RisqReplayControlButton } from './replay_control_button';

const PANEL_W = 420;
const PANEL_H = 44;
const PADDING = 8;
const GAP = 8;
const MARGIN = 12;
const TICK_MS = 400;
const INK = 'rgb(59, 36, 19)';

/** 0 is nothing processed, t - 0.5 is the intents for tick t, t is tick t processed */
function replayStops(tick_count: number): number[] {
  const ticks = Array.from({ length: tick_count }, (_, i: number): number[] => [i + 0.5, i + 1]);
  return [0, ...ticks.flat()];
}

/** Tan transport bar above the bottom panel: play/pause, a tick slider and exit */
export class RisqReplayPanel implements CanvasComponent {
  private shown = false;
  private playing = false;
  private tick_elapsed = 0;
  private hovering = false;
  private slider?: DwgRectSlider;
  private stops: number[] = [];
  private p: Point2D = { x: 0, y: 0 };
  private play_button = new RisqReplayControlButton(
    (): ReplayGlyph => (this.playing ? 'pause' : 'play'),
    (): void => this.togglePlaying()
  );
  private close_button = new RisqReplayControlButton(
    (): ReplayGlyph => 'close',
    (): void => this.risq.toggleReplay()
  );

  constructor(
    private risq: DwgRisq,
    private anchor: () => CanvasComponent
  ) {}

  open(tick_count: number): void {
    this.shown = true;
    this.playing = false;
    this.tick_elapsed = 0;
    this.stops = replayStops(tick_count);
    this.slider = new DwgRectSlider({
      slider_config: { fixed_values: this.stops, show_markers: true, value: 0 },
      p: { x: 0, y: 0 },
      w: 0,
      h: REPLAY_CONTROL_SIZE,
      draw_config: {
        fill_style: INK,
        hover_fill_style: 'rgb(90, 60, 35)',
        click_stroke_style: INK,
        stroke_width: 0,
        fixed_position: true,
      },
      track_draw_config: { fill_style: 'rgba(59, 36, 19, 0.3)', stroke_width: 0, fixed_position: true },
      track_height: 6,
      marker_color: 'rgba(59, 36, 19, 0.6)',
      onChange: (value: number): void => this.risq.showReplayTick(value),
    });
  }

  close(): void {
    this.shown = false;
    this.playing = false;
    this.hovering = false;
    for (const component of [this.play_button, this.close_button, this.slider]) {
      component?.setClicking(false);
      component?.setHovering(false);
    }
  }

  tick(): number {
    return this.slider?.value() ?? 0;
  }

  private togglePlaying(): void {
    if (!this.slider) {
      return;
    }
    this.playing = !this.playing;
    if (this.playing && this.slider.value() >= this.slider.maxValue()) {
      this.slider.setValue(this.slider.minValue());
    }
    this.risq.showMessage(this.playing ? 'Replay playing' : 'Replay paused', RISQ_MESSAGE_WARNING_COLOR);
  }

  private advance(dt: number): void {
    if (!this.playing || !this.slider) {
      return;
    }
    this.tick_elapsed += dt;
    while (this.tick_elapsed >= TICK_MS) {
      this.tick_elapsed -= TICK_MS;
      this.slider.setValue(this.stops[this.stops.indexOf(this.slider.value()) + 1] ?? this.slider.maxValue());
    }
    if (this.slider.value() >= this.slider.maxValue()) {
      this.togglePlaying();
    }
  }

  private layout(): void {
    const anchor = this.anchor();
    const canvas = this.risq.hudCanvasSize();
    const center_x = anchor.w() > 0 ? anchor.xi() + 0.5 * anchor.w() : 0.5 * canvas.width;
    const bottom = anchor.h() > 0 ? anchor.yi() : canvas.height - MARGIN;
    this.p = { x: center_x - 0.5 * PANEL_W, y: bottom - GAP - PANEL_H };
    const row_y = this.p.y + 0.5 * (PANEL_H - REPLAY_CONTROL_SIZE);
    const close_x = this.p.x + PANEL_W - PADDING - REPLAY_CONTROL_SIZE;
    const slider_x = this.p.x + PADDING + REPLAY_CONTROL_SIZE + GAP;
    this.play_button.setPosition({ x: this.p.x + PADDING, y: row_y });
    this.close_button.setPosition({ x: close_x, y: row_y });
    this.slider?.setPosition({ x: slider_x, y: row_y });
    this.slider?.setSize(close_x - GAP - slider_x, REPLAY_CONTROL_SIZE);
  }

  private controls(): CanvasComponent[] {
    return this.slider ? [this.play_button, this.slider, this.close_button] : [this.play_button, this.close_button];
  }

  isHovering(): boolean {
    return this.shown && this.hovering;
  }

  setHovering(hovering: boolean): void {
    this.hovering = this.hovering && hovering;
  }

  isClicking(): boolean {
    return this.shown && this.controls().some((control) => control.isClicking());
  }

  setClicking(clicking: boolean): void {
    if (!clicking) {
      this.controls().forEach((control) => control.setClicking(false));
    }
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    if (!this.shown) {
      return;
    }
    this.advance(dt);
    this.layout();
    const draw_config = {
      fill_style: 'rgb(222, 184, 135)',
      stroke_style: 'rgba(255, 255, 255, 0.2)',
      stroke_width: 1,
      fixed_position: true,
    };
    configDraw(ctx, transform, draw_config, false, false, () => {
      drawRect(ctx, this.p, PANEL_W, PANEL_H, 12);
    });
    this.controls().forEach((control) => control.draw(ctx, transform, dt));
  }

  mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    if (!this.shown) {
      return false;
    }
    this.layout();
    this.controls().forEach((control) => control.mousemove(canvas, screen, transform));
    this.hovering = screen.x >= this.xi() && screen.x <= this.xf() && screen.y >= this.yi() && screen.y <= this.yf();
    return this.hovering;
  }

  mousedown(e: MouseEvent): boolean {
    if (!this.shown) {
      return false;
    }
    const consumed = this.controls().map((control) => control.mousedown(e));
    return consumed.some(Boolean) || this.hovering;
  }

  mouseup(e: MouseEvent): void {
    if (this.shown) {
      this.controls().forEach((control) => control.mouseup(e));
    }
  }

  xi(): number {
    return this.p.x;
  }
  yi(): number {
    return this.p.y;
  }
  xf(): number {
    return this.p.x + PANEL_W;
  }
  yf(): number {
    return this.p.y + PANEL_H;
  }
  w(): number {
    return PANEL_W;
  }
  h(): number {
    return PANEL_H;
  }
}
