import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { DwgRectButton } from '../../../../util/canvas_components/button/rect_button';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import type { RisqMinimap } from '../minimap/risq_minimap';
import { MINIMAP_BUTTON_OVERFLOW } from '../minimap/corner_button';
import { RisqMercenaryGrid } from './mercenary_grid';
import type { RisqDefaultBehaviorGrid } from './default_behavior_grid';

export class RisqMercenaryPanel extends DwgRectButton {
  private open = false;
  private grid: RisqMercenaryGrid | RisqDefaultBehaviorGrid;
  constructor(
    private risq: DwgRisq,
    private minimap: () => RisqMinimap,
    grid?: RisqDefaultBehaviorGrid
  ) {
    super({
      button_config: {},
      p: { x: 0, y: 0 },
      w: 0,
      h: 0,
      r: 8,
      draw_config: {
        fill_style: 'rgb(222, 184, 135)',
        stroke_style: 'rgba(255, 255, 255, 0.2)',
        stroke_width: 1,
        fixed_position: true,
      },
      text: {
        text: '',
        config: {
          fill_style: 'black',
          align: 'center',
          baseline: 'middle',
          font: '12px serif',
        },
      },
    });
    this.grid = grid ?? new RisqMercenaryGrid(risq, 48);
  }
  private updateLayout(): void {
    this.grid.dataRefreshed();
    const side_width = Math.max(this.risq.left_panel.w(), this.risq.right_panel.w());
    const available_width = Math.max(0, this.risq.viewport.canvasSize().width - 2 * side_width - 32);
    this.grid.setAvailableWidth(available_width);
    const width = Math.min(available_width + 16, Math.max(160, this.grid.w() + 16));
    const height = (this.grid.h() || 32) + 16;
    const minimap = this.minimap();
    const p = { x: minimap.xc() - width / 2, y: Math.max(8, minimap.yi() - MINIMAP_BUTTON_OVERFLOW - 8 - height) };
    this.setText(this.grid.h() ? '' : 'No mercenaries available');
    if (width === this.w() && height === this.h() && p.x === this.xi() && p.y === this.yi()) {
      return;
    }
    this.setSize(width, height);
    this.setPosition(p);
    this.grid.setPosition({ x: p.x + 8, y: p.y + 8 });
  }

  toggle(open?: boolean): void {
    this.open = open ?? !this.open;
    this.setHovering(false);
    this.setClicking(false);
    if (this.open) {
      this.updateLayout();
    }
    this.risq.pointer.recalculate();
  }
  override draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    if (!this.open) {
      return;
    }
    this.updateLayout();
    super.draw(ctx, transform, dt);
    this.grid.draw(ctx, transform, dt);
    this.grid.drawTooltip(ctx, transform, this.risq, dt);
  }
  override mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    if (!this.open) {
      return false;
    }
    const hovering = super.mousemove(canvas, screen, transform);
    if (hovering) {
      this.grid.mousemove(canvas, screen, transform);
    } else {
      this.grid.setHovering(false);
    }
    return hovering;
  }
  override mousedown(e: MouseEvent): boolean {
    return this.open && (this.grid.mousedown(e) || this.isHovering());
  }
  override mouseup(e: MouseEvent): void {
    if (this.open) {
      this.grid.mouseup(e);
    }
  }
  override isClicking(): boolean {
    return this.open && this.grid.isClicking();
  }
  override setHovering(hovering: boolean): void {
    super.setHovering(hovering);
    if (!hovering) {
      this.grid.setHovering(false);
    }
  }
  override setClicking(clicking: boolean): void {
    super.setClicking(clicking);
    this.grid.setClicking(clicking);
  }
  scroll(dy: number, mode: number): boolean {
    return this.open && this.grid.scroll(dy, mode);
  }
  protected hovered(): void {}
  protected unhovered(): void {}
  protected clicked(): void {}
  protected released(): void {}
}
