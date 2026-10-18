import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { ClickSource } from '../../../../util/canvas_components/button/button_config';
import { DwgRectButton } from '../../../../util/canvas_components/button/rect_button';
import { DwgSquareButton } from '../../../../util/canvas_components/button/square_button';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawLine, drawRect } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import type { IdleCategory } from '../../application/orders/orders_model';
import type { DwgRisq } from '../../risq';
import { RisqMinimapCornerButton } from '../minimap/corner_button';
import { RisqActionButton } from '../left_panel/actions/action_button';
import { RISQ_MESSAGE_WARNING_COLOR } from '../message_queue';
import { getSettings } from '../../../../../../scripts/settings_store';
import { RisqHotkeyAction } from '../../application/input/hotkeys';
import { techImage } from '../../rendering/assets/buildings';

export class RisqBottomPanelButton extends DwgSquareButton {
  constructor(private risq: DwgRisq) {
    super({
      button_config: { allow_nonleft_clicks: false },
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
      rotation: Math.PI,
    });
  }

  protected hovered(): void {}
  protected unhovered(): void {}
  protected clicked(): void {}
  protected released(): void {
    if (!this.isHovering()) {
      return;
    }
    this.risq.bottom_panel.toggle();
    this.setHovering(false);
  }
}

export class RisqMercenaryPanelButton extends RisqMinimapCornerButton {
  constructor(private risq: DwgRisq) {
    super(risq, { row: 0, col: 0, image_path: techImage(4), description: 'Mercenaries' }, Math.PI);
  }
  protected released(): void {
    if (this.isHovering()) {
      this.risq.toggleMercenaryPanel();
    }
  }
}

export class RisqDefaultBehaviorButton extends RisqMinimapCornerButton {
  constructor(private risq: DwgRisq) {
    super(
      risq,
      { row: 0, col: 0, image_path: 'icons/unit64', description: 'New military unit behavior' },
      -Math.PI / 2
    );
  }
  protected released(): void {
    if (this.isHovering()) {
      this.risq.toggleDefaultBehaviorPanel();
    }
  }
}

export class RisqSummaryReportButton extends RisqActionButton {
  private risq: DwgRisq;

  constructor(risq: DwgRisq) {
    super({ row: 0, col: 0, image_path: 'icons/report64', description: 'Review last turn report' }, 30);
    this.risq = risq;
  }

  override dataRefreshed(): void {
    this.dimmed = !this.risq.session.getLastTurnReport();
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[RisqHotkeyAction.SUMMARY_REPORT]);
  }

  protected released(): void {
    if (!this.isHovering()) {
      return;
    }
    if (this.dimmed) {
      this.risq.showMessage('No previous turn report to review', RISQ_MESSAGE_WARNING_COLOR);
    } else {
      this.risq.reviewLastTurnReport();
    }
  }
}

export class RisqViewModeButton extends RisqMinimapCornerButton {
  private risq: DwgRisq;

  constructor(risq: DwgRisq) {
    super(risq, { row: 0, col: 0, image_path: 'icons/eye64', description: 'Toggle view mode' }, 0);
    this.risq = risq;
  }

  override dataRefreshed(): void {
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[RisqHotkeyAction.CYCLE_VIEW_MODE]);
  }

  protected released(): void {
    if (this.isHovering()) {
      this.risq.viewport.cycleViewMode();
    }
  }
}

export class RisqTechTreeButton extends RisqMinimapCornerButton {
  private risq: DwgRisq;

  constructor(risq: DwgRisq) {
    super(risq, { row: 0, col: 0, image_path: 'icons/research64', description: 'Open tech tree' }, Math.PI / 2);
    this.risq = risq;
  }

  override dataRefreshed(): void {
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[RisqHotkeyAction.TECH_TREE]);
  }

  protected released(): void {
    if (this.isHovering()) {
      this.risq.openTechTree();
    }
  }
}

export class RisqControlGroupCloseButton extends DwgRectButton {
  constructor(
    p: Point2D,
    size: number,
    private on_remove: () => void
  ) {
    super({
      button_config: {},
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
      text: {
        text: '×',
        config: { fill_style: 'black', align: 'center', baseline: 'middle', font: 'bold 14px serif' },
      },
    });
  }
  protected hovered(): void {}
  protected unhovered(): void {}
  protected clicked(): void {}
  protected released(source: ClickSource): void {
    if (source === ClickSource.LEFT_MOUSE && this.isHovering()) {
      this.on_remove();
    }
  }
}

export const IDLE_FILTERS: [IdleCategory, string, string][] = [
  ['economic', 'icons/villager64', 'Idle economic units'],
  ['military', 'icons/unit64', 'Idle military units'],
  ['unit_production', 'icons/building64', 'Idle unit-producing buildings'],
  ['other_buildings', 'icons/research64', 'Idle non-unit-producing buildings'],
];

export class RisqIdleFilterButton extends RisqActionButton {
  constructor(
    private risq: DwgRisq,
    readonly category: IdleCategory,
    image_path: string,
    description: string,
    col: number
  ) {
    super({ row: 0, col, image_path, description }, 28);
  }
  override execute(): void {
    this.risq.toggleIdleCategory(this.category);
  }
  override draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    super.draw(ctx, transform, dt);
    configDraw(
      ctx,
      transform,
      { fill_style: 'white', stroke_style: 'black', stroke_width: 1, fixed_position: true },
      false,
      false,
      (): void => {
        const p = { x: this.xi(), y: this.yi() };
        const size = Math.min(10, this.w());
        drawRect(ctx, p, size, size);
        if (this.risq.planning.idleCategoryEnabled(this.category)) {
          drawLine(ctx, { x: p.x + size * 0.2, y: p.y + size * 0.5 }, { x: p.x + size * 0.4, y: p.y + size * 0.8 });
          drawLine(ctx, { x: p.x + size * 0.4, y: p.y + size * 0.8 }, { x: p.x + size * 0.8, y: p.y + size * 0.2 });
        }
      }
    );
  }
}
