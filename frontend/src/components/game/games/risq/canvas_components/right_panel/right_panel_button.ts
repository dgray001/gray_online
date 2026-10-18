import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { DwgSquareButton } from '../../../../util/canvas_components/button/square_button';
import type { DwgRisq } from '../../risq';
import { RisqActionButton } from '../left_panel/actions/action_button';
import { RISQ_MESSAGE_WARNING_COLOR } from '../message_queue';
import { RisqButtonTooltip } from '../risq_tooltip';

export class RisqRightPanelButton extends DwgSquareButton {
  private risq: DwgRisq;

  constructor(risq: DwgRisq) {
    super({
      button_config: {
        allow_nonleft_clicks: false,
      },
      p: { x: 0, y: 0 },
      s: 36,
      draw_config: {
        fill_style: 'transparent',
        stroke_style: 'transparent',
        stroke_width: 0,
        hover_fill_style: 'rgb(180, 180, 180, 0.5)',
        click_fill_style: 'rgb(210, 210, 210, 0.7)',
        fixed_position: true,
      },
      move_animation_speed: 200,
      rotate_animation_speed: 200,
      image_path: 'icons/triangle_gray36',
      rotation: -0.5 * Math.PI,
    });
    this.risq = risq;
  }

  protected hovered(): void {}
  protected unhovered(): void {}
  protected clicked(): void {}
  protected released(): void {
    if (!this.isHovering()) {
      return;
    }
    this.risq.right_panel.toggle();
    this.setHovering(false);
  }
}

export class RisqReplayButton extends RisqActionButton {
  constructor(private risq: DwgRisq) {
    super(
      { row: 0, col: 0, image_path: 'icons/refresh_black32', description: 'Replay last turn', image_on_top: true },
      30
    );
  }

  override isClicking(): boolean {
    return super.isClicking() || this.risq.session.isReplaying();
  }

  override dataRefreshed(): void {
    this.dimmed = !this.risq.getGame()?.last_turn_journal && !this.risq.session.isReplaying();
  }
  protected released(): void {
    if (!this.isHovering()) {
      return;
    }
    if (this.dimmed) {
      this.risq.showMessage('No previous turn to replay', RISQ_MESSAGE_WARNING_COLOR);
    } else {
      this.risq.toggleReplay();
    }
  }
}

export class RisqOrdersExpandButton extends DwgSquareButton {
  private expanded = false;
  private tooltip = new RisqButtonTooltip(() => ({
    title: this.expanded ? 'Collapse orders list' : 'Expand orders list',
  }));

  constructor(private risq: DwgRisq) {
    super({
      button_config: { allow_nonleft_clicks: false },
      p: { x: 0, y: 0 },
      s: 22,
      draw_config: {
        fill_style: 'transparent',
        stroke_style: 'transparent',
        stroke_width: 0,
        hover_fill_style: 'rgba(210, 180, 130, 0.5)',
        click_fill_style: 'rgba(235, 210, 165, 0.7)',
        draw_clicked_when_unhovered: true,
        fixed_position: true,
      },
      image_path: 'icons/expand64',
      image_on_top: true,
    });
  }

  setExpanded(expanded: boolean): void {
    this.expanded = expanded;
    this.setImage(this.risq.getIcon(expanded ? 'icons/collapse64' : 'icons/expand64'));
  }

  protected hovered(): void {}
  protected unhovered(): void {}
  protected clicked(): void {}
  protected released(): void {
    if (!this.isHovering()) {
      return;
    }
    this.risq.right_panel.toggleOrdersExpanded();
    this.setHovering(false);
  }

  drawTooltip(ctx: CanvasRenderingContext2D, transform: BoardTransformData, risq: DwgRisq, dt: number): void {
    this.tooltip.draw(ctx, transform, risq, dt, this.isHovering(), this.isPressed());
  }
}
