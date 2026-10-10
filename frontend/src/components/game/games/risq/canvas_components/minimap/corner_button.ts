import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import { RisqActionButton } from '../left_panel/actions/action_button';
import type { RisqActionButtonConfig } from '../left_panel/actions/action_button';
import { rotatePoint } from '../../../../util/objects2d';

export const MINIMAP_BUTTON_OVERFLOW = 8;

export abstract class RisqMinimapCornerButton extends RisqActionButton {
  private icon: CanvasImageSource;
  private circle_center: Point2D = { x: 0, y: 0 };
  private inner_radius = 0;
  private minimap_side = 0;
  private arc_angle = 0;

  constructor(
    risq: DwgRisq,
    config: RisqActionButtonConfig,
    private corner_rotation: number
  ) {
    super(config, 0);
    this.icon = risq.getIcon(config.image_path);
  }

  setMinimapBounds(p: Point2D, side: number): void {
    const size = Math.min(56, side / 2);
    const outer_radius = side / 2 + MINIMAP_BUTTON_OVERFLOW;
    const offset = rotatePoint({ x: outer_radius - size / 2, y: outer_radius - size / 2 }, this.corner_rotation);
    const corner = {
      x: p.x + side / 2 + offset.x - size / 2,
      y: p.y + side / 2 + offset.y - size / 2,
    };
    if (this.minimap_side === side && this.w() === size && this.xi() === corner.x && this.yi() === corner.y) {
      return;
    }
    this.setSize(size, size);
    this.minimap_side = side;
    this.setPosition(corner);
    this.circle_center = { x: p.x + side / 2, y: p.y + side / 2 };
    this.inner_radius = side / 2 + 4;
    this.arc_angle = Math.asin((outer_radius - size) / this.inner_radius);
  }
  override mouseOver(canvas: Point2D, screen: Point2D): boolean {
    return (
      this.w() > 0 &&
      super.mouseOver(canvas, screen) &&
      Math.hypot(screen.x - this.circle_center.x, screen.y - this.circle_center.y) >= this.inner_radius
    );
  }
  protected override _draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, _dt: number): void {
    if (this.w() <= 0) {
      return;
    }
    configDraw(
      ctx,
      transform,
      {
        fill_style: 'transparent',
        stroke_width: 0,
        hover_fill_style: 'rgba(220, 220, 220, 0.2)',
        click_fill_style: 'rgba(250, 250, 250, 0.4)',
        fixed_position: true,
        draw_clicked_when_unhovered: true,
      },
      this.isHovering(),
      this.isClicking(),
      () => {
        this.drawRotatedCorner(ctx);
        this.drawIcon(ctx);
      }
    );
  }
  protected override drawDimmedShape(ctx: CanvasRenderingContext2D): void {
    this.drawRotatedCorner(ctx);
  }
  private drawRotatedCorner(ctx: CanvasRenderingContext2D): void {
    ctx.save();
    ctx.translate(this.circle_center.x, this.circle_center.y);
    ctx.rotate(this.corner_rotation);
    this.drawCorner(ctx);
    ctx.restore();
  }
  private drawCorner(ctx: CanvasRenderingContext2D): void {
    const radius = this.inner_radius;
    const outer = this.minimap_side / 2 + MINIMAP_BUTTON_OVERFLOW;
    const inner = outer - this.w();
    ctx.moveTo(outer, inner);
    ctx.lineTo(outer, outer);
    ctx.lineTo(inner, outer);
    ctx.lineTo(inner, radius * Math.cos(this.arc_angle));
    ctx.arc(0, 0, radius, Math.PI / 2 - this.arc_angle, this.arc_angle, true);
    ctx.closePath();
    ctx.fill();
  }
  private drawIcon(ctx: CanvasRenderingContext2D): void {
    const size = 28;
    const offset = this.minimap_side / 2 + MINIMAP_BUTTON_OVERFLOW - size / 2 - 2;
    const center = rotatePoint({ x: offset, y: offset }, this.corner_rotation);
    ctx.drawImage(
      this.icon,
      this.circle_center.x + center.x - size / 2,
      this.circle_center.y + center.y - size / 2,
      size,
      size
    );
  }
}
