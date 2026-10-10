import { DwgRectButton } from '../../../../util/canvas_components/button/rect_button';
import type { Point2D } from '../../../../util/objects2d';
import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawUnitCountBadge } from '../../rendering/unit_count_badge';
import { drawCircle } from '../../../../util/canvas_util';
import type { ControlGroup } from '../../application/selection/control_groups';

export class RisqControlGroupIcon extends DwgRectButton {
  private selected = false;
  private pressed_button: number | undefined;
  constructor(
    private icon: CanvasImageSource | (() => CanvasImageSource),
    p: Point2D,
    size: number,
    private count: number,
    private subject: ControlGroup,
    private on_click: (event: MouseEvent) => void
  ) {
    super({
      button_config: { allow_nonleft_clicks: true },
      p,
      w: size,
      h: size,
      draw_config: {
        fill_style: 'transparent',
        stroke_width: 0,
        fixed_position: true,
        hover_fill_style: 'rgba(220, 220, 220, 0.2)',
        click_fill_style: 'rgba(250, 250, 250, 0.4)',
      },
    });
    this.setImage(typeof icon === 'function' ? icon() : icon);
  }
  setSelection(unit_ids: ReadonlySet<number>, building_ids: ReadonlySet<number>): void {
    this.selected =
      this.subject.kind === 'building'
        ? this.subject.ids.some((id: number): boolean => building_ids.has(id))
        : this.subject.ids.some((id) => unit_ids.has(id));
  }
  protected override _draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    if (typeof this.icon === 'function') {
      this.setImage(this.icon());
    }
    super._draw(ctx, transform, dt);
    if (this.selected) {
      configDraw(
        ctx,
        transform,
        {
          fill_style: 'transparent',
          stroke_style: 'white',
          stroke_width: 1.5,
          fixed_position: true,
        },
        false,
        false,
        () => {
          drawCircle(ctx, { x: this.xc(), y: this.yc() }, this.w() / 2);
        }
      );
    }
    if (this.count > 1) {
      configDraw(
        ctx,
        transform,
        { fill_style: 'transparent', stroke_width: 0, fixed_position: true },
        false,
        false,
        () => {
          drawUnitCountBadge(ctx, this.count, { x: this.xi(), y: this.yi() }, this.w());
        }
      );
    }
  }
  protected hovered(): void {}
  override mousedown(e: MouseEvent): boolean {
    const accepted = (e.button === 0 || e.button === 2) && super.mousedown(e);
    if (accepted) {
      this.pressed_button = e.button;
    }
    return accepted;
  }
  override mouseup(e: MouseEvent): void {
    const activate = e.button === this.pressed_button && this.isClicking() && this.isHovering();
    super.mouseup(e);
    this.pressed_button = undefined;
    if (activate) {
      this.on_click(e);
    }
  }
  protected unhovered(): void {}
  protected clicked(): void {}
  protected released(): void {}
}
