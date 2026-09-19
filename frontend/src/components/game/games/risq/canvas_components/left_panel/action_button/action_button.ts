import type { BoardTransformData } from '../../../../../util/canvas_board/canvas_board';
import { DwgSquareButton } from '../../../../../util/canvas_components/button/square_button';
import { configDraw } from '../../../../../util/canvas_components/canvas_component';
import { drawRect } from '../../../../../util/canvas_util';
import {
  createTooltipState,
  drawTooltip as drawGenericTooltip,
  shouldShowTooltip,
} from '../../../../../util/canvas_components/tooltip';
import type { DwgRisq } from '../../../risq';
import type { RisqTooltipData } from '../../risq_tooltip';
import { createRisqTooltipState, drawRisqTooltip } from '../../risq_tooltip';

export declare interface RisqActionButtonConfig {
  row: number;
  col: number;
  image_path: string;
  description: string;
}

export abstract class RisqActionButton extends DwgSquareButton {
  readonly row: number;
  readonly col: number;
  readonly description: string;
  protected tooltip_state = createTooltipState();
  protected dimmed = false;

  constructor(config: RisqActionButtonConfig, s: number) {
    super({
      button_config: {},
      p: { x: 0, y: 0 },
      s,
      draw_config: {
        fill_style: 'transparent',
        stroke_style: 'rgb(60, 40, 20)',
        stroke_width: 1,
        hover_fill_style: 'rgba(210, 180, 130, 0.5)',
        click_fill_style: 'rgba(235, 210, 165, 0.7)',
        draw_clicked_when_unhovered: true,
        fixed_position: true,
      },
      image_path: config.image_path,
    });
    this.row = config.row;
    this.col = config.col;
    this.description = config.description;
  }

  protected hovered(): void {}
  protected unhovered(): void {}
  protected clicked(): void {}

  dataRefreshed(): void {}

  override draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    super.draw(ctx, transform, dt);
    if (this.isDisabled() || this.dimmed) {
      configDraw(
        ctx,
        transform,
        { fill_style: 'rgba(40, 30, 20, 0.55)', stroke_width: 0, fixed_position: true },
        false,
        false,
        () => drawRect(ctx, { x: this.xi(), y: this.yi() }, this.w(), this.h())
      );
    }
  }

  protected getTooltipData(): RisqTooltipData {
    return { title: this.description };
  }

  drawTooltip(ctx: CanvasRenderingContext2D, transform: BoardTransformData, risq: DwgRisq, dt: number) {
    if (!shouldShowTooltip(this.tooltip_state, this.isHovering(), this.isClicking(), dt)) {
      return;
    }
    drawGenericTooltip(
      this.tooltip_state,
      ctx,
      transform,
      risq.canvasSize(),
      { x: this.xi(), y: this.yi() },
      this.getTooltipData().title
    );
  }
}

/** Base for action buttons whose tooltip needs cost/stamina/description, not just a plain title */
export abstract class RisqRichTooltipActionButton extends RisqActionButton {
  private rich_tooltip_state = createRisqTooltipState();

  override drawTooltip(ctx: CanvasRenderingContext2D, transform: BoardTransformData, risq: DwgRisq, dt: number) {
    drawRisqTooltip(
      ctx,
      transform,
      risq,
      dt,
      this.isHovering(),
      this.isClicking(),
      this.rich_tooltip_state,
      { x: this.xi(), y: this.yi() },
      this.getTooltipData()
    );
  }
}
