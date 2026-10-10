import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { DwgRectButton } from '../../../../util/canvas_components/button/rect_button';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawText } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import { STANCES, TOGGLES } from '../left_panel/actions/action_factory';
import type { RisqActionButton } from '../left_panel/actions/action_button';
import { RisqStanceButton } from '../left_panel/actions/unit/stance_button';
import { RisqUnitToggleButton } from '../left_panel/actions/unit/unit_toggle_button';
import { RisqTargetPriorityControl } from '../left_panel/controls/target_priority_control';

export class RisqDefaultBehaviorGrid extends DwgRectButton {
  private buttons: RisqActionButton[];
  private priority: RisqTargetPriorityControl;
  constructor(private risq: DwgRisq) {
    super({
      button_config: {},
      p: { x: 0, y: 0 },
      w: 264,
      h: 180,
      draw_config: { fill_style: 'transparent', stroke_width: 0, fixed_position: true },
    });
    this.buttons = STANCES.map(
      ([stance, image_path, description], col): RisqActionButton =>
        new RisqStanceButton(
          { stance, image_path, description, row: 0, col, unit_internal_ids: [], defaults: true },
          risq,
          36
        )
    );
    this.priority = new RisqTargetPriorityControl(risq, [], true);
    this.buttons.push(
      ...TOGGLES.map(
        ([field, image_path, description], col): RisqActionButton =>
          new RisqUnitToggleButton(
            { field, image_path, description, row: 1, col, unit_internal_ids: [], defaults: true },
            risq,
            36
          )
      )
    );
  }
  setAvailableWidth(width: number): void {
    this.setSize(Math.min(264, width), 180);
  }
  dataRefreshed(): void {
    for (const button of this.buttons) {
      button.dataRefreshed();
      if (this.risq.session.canGiveOrders()) {
        button.enable();
      } else {
        button.disable();
      }
    }
    this.priority.dataRefreshed();
    if (!this.risq.session.canGiveOrders()) {
      this.priority.cancelInput();
    }
  }
  override setPosition(p: Point2D): void {
    super.setPosition(p);
    const size = Math.max(0, Math.min(36, (this.w() - 32) / 4));
    for (const button of this.buttons) {
      button.setSize(size, size);
      button.setPosition({ x: p.x + button.col * (size + 8), y: p.y + 24 + button.row * 44 });
    }
    this.priority.setPosition({ x: p.x, y: p.y + 132 });
    this.priority.setSize(this.w(), size + 8);
  }
  override draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    super.draw(ctx, transform, dt);
    configDraw(
      ctx,
      transform,
      { fill_style: 'transparent', stroke_width: 0, fixed_position: true },
      false,
      false,
      (): void => {
        drawText(ctx, 'New military units', {
          p: { x: this.xi(), y: this.yi() },
          w: this.w(),
          fill_style: 'black',
          font: 'bold 16px serif',
        });
        drawText(ctx, 'Target priority', {
          p: { x: this.xi(), y: this.yi() + 112 },
          w: this.w(),
          fill_style: 'black',
          font: '14px serif',
        });
      }
    );
    for (const button of this.buttons) {
      button.draw(ctx, transform, dt);
    }
    this.priority.draw(ctx, transform);
  }
  drawTooltip(ctx: CanvasRenderingContext2D, transform: BoardTransformData, risq: DwgRisq, dt: number): void {
    for (const button of this.buttons) {
      button.drawTooltip(ctx, transform, risq, dt);
    }
  }
  override mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    const hovering = super.mousemove(canvas, screen, transform);
    for (const button of this.buttons) {
      button.mousemove(canvas, screen, transform);
    }
    this.priority.mousemove(screen);
    return hovering;
  }
  override mousedown(e: MouseEvent): boolean {
    if (!this.risq.session.canGiveOrders() || e.button !== 0) {
      return this.isHovering();
    }
    const priority = this.priority.mousedown(e);
    return this.buttons.map((button): boolean => button.mousedown(e)).some(Boolean) || priority || this.isHovering();
  }
  override mouseup(e: MouseEvent): void {
    for (const button of this.buttons) {
      button.mouseup(e);
    }
    if (this.risq.session.canGiveOrders() && e.button === 0) {
      this.priority.mouseup(e);
    }
  }
  override setClicking(clicking: boolean): void {
    for (const button of this.buttons) {
      button.setClicking(clicking);
    }
    if (!clicking) {
      this.priority.cancelInput();
    }
  }
  override setHovering(hovering: boolean): void {
    super.setHovering(hovering);
    if (!hovering) {
      this.buttons.forEach((button): void => button.setHovering(false));
    }
  }
  override isClicking(): boolean {
    return this.priority.isClicking() || this.buttons.some((button): boolean => button.isPressed());
  }
  scroll(_dy: number, _mode: number): boolean {
    return false;
  }
  protected hovered(): void {}
  protected unhovered(): void {}
  protected clicked(): void {}
  protected released(): void {}
}
