import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { drawRect, drawText } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import { ClickSource } from '../../../../util/canvas_components/button/button_config';
import { DwgRectButton } from '../../../../util/canvas_components/button/rect_button';
import type { DrawConfig } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import type { RisqLeftPanel } from './left_panel';

export declare interface SpaceUnitsRowButtonConfig {
  p: Point2D;
  w: number;
  h: number;
  icon: HTMLImageElement;
  panel: RisqLeftPanel;
  economic: boolean;
}

/** A full-width clickable row in the space panel showing an icon and a count, e.g. "villagers in this space" */
export class RisqSpaceUnitsRowButton extends DwgRectButton {
  private icon: HTMLImageElement;
  private panel: RisqLeftPanel;
  private economic: boolean;
  private row_text = '';
  private draw_config: DrawConfig;

  constructor(config: SpaceUnitsRowButtonConfig) {
    const draw_config: DrawConfig = {
      fill_style: 'transparent',
      stroke_style: 'transparent',
      stroke_width: 0,
      hover_fill_style: 'rgba(210, 210, 210, 0.25)',
      click_fill_style: 'rgba(250, 250, 250, 0.4)',
      draw_clicked_when_unhovered: true,
      fixed_position: true,
    };
    super({
      button_config: {},
      p: config.p,
      w: config.w,
      h: config.h,
      draw_config,
    });
    this.draw_config = draw_config;
    this.icon = config.icon;
    this.panel = config.panel;
    this.economic = config.economic;
  }

  setRowText(text: string): void {
    this.row_text = text;
  }

  protected override _draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, _dt: number): void {
    configDraw(ctx, transform, this.draw_config, this.isHovering(), this.isClicking(), () => {
      drawRect(ctx, { x: this.xi(), y: this.yi() }, this.w(), this.h());
      ctx.drawImage(this.icon, this.xi(), this.yi(), this.h(), this.h());
      drawText(ctx, this.row_text, {
        p: { x: this.xi() + this.h() + 2, y: this.yi() },
        w: this.w() - this.h() - 2,
        fill_style: 'black',
        align: 'left',
        baseline: 'top',
        font: `bold ${this.h()}px serif`,
      });
    });
  }

  protected hovered(): void {}
  protected unhovered(): void {}
  protected clicked(): void {}

  protected released(source: ClickSource): void {
    if (source === ClickSource.LEFT_MOUSE && this.isHovering()) {
      this.panel.openSpaceUnitsRow(this.economic);
    }
  }
}
