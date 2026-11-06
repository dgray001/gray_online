import type { BoardTransformData } from '../../../../../util/canvas_board/canvas_board';
import { configDraw } from '../../../../../util/canvas_components/canvas_component';
import { DwgRectSlider } from '../../../../../util/canvas_components/slider/rect_slider';
import type { Point2D } from '../../../../../util/objects2d';
import type { DwgRisq } from '../../../risq';

export const ATTACK_STACK_STOPS = [1, 2, 3, 4, 6, 8, 10, 15, 0] as const;

export function attackStackLabel(val: number): string {
  return val === 0 ? '∞' : `${val}`;
}

function nearestStopIndex(val: number): number {
  if (val === 0) {
    return ATTACK_STACK_STOPS.length - 1;
  }
  let best_idx = 0;
  let best_diff = Math.abs(ATTACK_STACK_STOPS[0] - val);
  for (let i = 1; i < ATTACK_STACK_STOPS.length - 1; i++) {
    const diff = Math.abs(ATTACK_STACK_STOPS[i] - val);
    if (diff < best_diff) {
      best_diff = diff;
      best_idx = i;
    }
  }
  return best_idx;
}

export class RisqAttackStackControl extends DwgRectSlider {
  private p: Point2D = { x: 0, y: 0 };
  private control_w = 0;
  private control_h = 0;
  private current_stack = 0;
  private mixed = false;
  private row_hovering = false;
  private syncing = false;

  constructor(
    private risq: DwgRisq,
    private unit_internal_ids: number[],
    private defaults: boolean = false
  ) {
    const fixed_indices = ATTACK_STACK_STOPS.map((_, i) => i);
    super({
      slider_config: {
        fixed_values: fixed_indices,
        show_markers: true,
        value: fixed_indices.length - 1,
      },
      p: { x: 0, y: 0 },
      w: 100,
      h: 24,
      draw_config: {
        fill_style: 'rgb(95, 62, 35)',
        stroke_style: 'black',
        stroke_width: 1.5,
        hover_fill_style: 'rgb(125, 85, 52)',
        hover_stroke_style: 'black',
        click_fill_style: 'rgb(75, 48, 26)',
        click_stroke_style: 'rgb(59, 36, 19)',
        fixed_position: true,
      },
      track_draw_config: {
        fill_style: 'rgba(59, 36, 19, 0.3)',
        stroke_style: 'rgba(59, 36, 19, 0.7)',
        stroke_width: 1,
        fixed_position: true,
      },
      track_height: 4,
      thumb_width: 12,
      thumb_height: 18,
      marker_color: 'black',
      marker_size: 8,
      onChange: (index: number): void => this.onSliderChange(Math.round(index)),
    });
  }

  setUnitIds(unit_internal_ids: number[]): void {
    this.unit_internal_ids = unit_internal_ids;
  }

  unitIds(): number[] {
    return this.unit_internal_ids;
  }

  override setPosition(p: Point2D): void {
    this.p = { ...p };
    this.updateSliderLayout();
  }

  override setSize(w: number, h: number): void {
    this.control_w = w;
    this.control_h = h;
    this.updateSliderLayout();
  }

  private updateSliderLayout(): void {
    const pad_x = 4;
    const slider_h = 18;
    const slider_x = this.p.x + pad_x;
    const slider_y = this.p.y + Math.max(0, this.control_h - slider_h);
    const slider_w = Math.max(40, this.control_w - 2 * pad_x);
    super.setPosition({ x: slider_x, y: slider_y });
    super.setSize(slider_w, slider_h);
  }

  dataRefreshed(): void {
    if (this.isClicking()) {
      return;
    }
    const player = this.risq.getPlayer();
    const stacks = this.defaults
      ? [player?.default_unit_attack_stack ?? 0]
      : this.unit_internal_ids.map((id) => player?.units.get(id)?.attack_stack ?? 0);
    const first = stacks[0] ?? 0;
    this.mixed = stacks.some((v) => v !== first);
    this.current_stack = this.mixed ? 0 : first;
    const idx = nearestStopIndex(this.current_stack);
    this.syncing = true;
    try {
      this.setValue(idx);
    } finally {
      this.syncing = false;
    }
    if (this.risq.session.canGiveOrders()) {
      this.enable();
    } else {
      this.disable();
    }
  }

  private onSliderChange(index: number): void {
    if (this.syncing) {
      return;
    }
    const safe_idx = Math.max(0, Math.min(ATTACK_STACK_STOPS.length - 1, index));
    this.current_stack = ATTACK_STACK_STOPS[safe_idx];
    this.mixed = false;
  }

  protected override released(): void {
    super.released();
    this.commitChange();
  }

  private commitChange(): void {
    if (this.defaults) {
      this.risq.commands.setDefaultUnitBehavior({ attack_stack: this.current_stack });
    } else {
      this.risq.commands.setUnitAttackStack(this.unit_internal_ids, this.current_stack);
    }
  }

  override isHovering(): boolean {
    return this.row_hovering || super.isHovering();
  }

  override setHovering(hovering: boolean): void {
    this.row_hovering = hovering;
    super.setHovering(hovering);
  }

  cancelInput(): void {
    this.setClicking(false);
    this.setHovering(false);
  }

  override draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number = 0): void {
    configDraw(ctx, transform, { fill_style: 'black', stroke_width: 0, fixed_position: true }, false, false, () => {
      const x = this.p.x + 4;
      const y = this.p.y + Math.max(0, this.control_h - 18) - 2;
      ctx.fillStyle = 'black';
      ctx.textAlign = 'left';
      ctx.textBaseline = 'bottom';
      ctx.font = 'bold 20px serif';
      const label_w = ctx.measureText('Stack: ').width;
      if (this.current_stack === 0 && !this.mixed) {
        ctx.fillText('Stack: ', x, y);
        ctx.font = 'bold 26px serif';
        ctx.fillText('∞', x + label_w, y);
      } else {
        ctx.fillText(`Stack: ${this.mixed ? '?' : this.current_stack}`, x, y);
      }
    });
    super.draw(ctx, transform, dt);
  }

  override mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    const in_bounds =
      screen.x >= this.p.x &&
      screen.x <= this.p.x + this.control_w &&
      screen.y >= this.p.y &&
      screen.y <= this.p.y + this.control_h;
    this.row_hovering = in_bounds;
    return super.mousemove(canvas, screen, transform) || in_bounds;
  }

  protected override mouseOver(_canvas: Point2D, screen: Point2D): boolean {
    return (
      screen.x >= this.p.x &&
      screen.x <= this.p.x + this.control_w &&
      screen.y >= this.p.y &&
      screen.y <= this.p.y + this.control_h
    );
  }

  override mousedown(e: MouseEvent): boolean {
    return super.mousedown(e);
  }

  override mouseup(e: MouseEvent): void {
    super.mouseup(e);
  }

  override xi(): number {
    return this.p.x;
  }
  override xf(): number {
    return this.p.x + this.control_w;
  }
  override yi(): number {
    return this.p.y;
  }
  override yf(): number {
    return this.p.y + this.control_h;
  }
  override w(): number {
    return this.control_w;
  }
  override h(): number {
    return this.control_h;
  }
}
