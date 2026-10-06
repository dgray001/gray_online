import { drawHexagon } from '../../game/util/canvas_util';
import type { Point2D } from '../../game/util/objects2d';
import type { EditorConfigPanel, EditorTarget, EditorToolContext } from '../editor_tool';
import { EditorTool } from '../editor_tool';
import type { ValidationIssue } from './validation_panel';
import { ValidationPanel } from './validation_panel';

export class ValidationTool extends EditorTool {
  readonly id = 'validate';
  readonly label = 'Validate';
  readonly hotkey = 'x';
  private panel: ValidationPanel;
  private highlighted_space?: Point2D;
  constructor(context: EditorToolContext) {
    super(context);
    this.panel = new ValidationPanel(context, (issue) => this.selectIssue(issue));
  }

  config(): EditorConfigPanel {
    return this.panel;
  }

  activated(): void {
    this.panel.refresh();
  }

  deactivated(): void {
    this.highlighted_space = undefined;
  }

  private selectIssue(issue: ValidationIssue): void {
    this.highlighted_space = issue.space;
  }
  mousedown(_target: EditorTarget | undefined, _e: MouseEvent): boolean {
    return false;
  }
  mousemove(_target: EditorTarget | undefined): void {}
  mouseup(_target: EditorTarget | undefined, _e: MouseEvent): void {}

  drawOverlay(ctx: CanvasRenderingContext2D, _target: EditorTarget | undefined): void {
    if (!this.highlighted_space) {
      return;
    }
    const canvas_pt = this.context.toCanvas(this.highlighted_space);
    ctx.strokeStyle = 'rgba(255, 60, 60, 0.9)';
    ctx.lineWidth = 3;
    drawHexagon(ctx, canvas_pt, this.context.hexR());
  }
}
