import { drawHexagon } from '../../game/util/canvas_util';
import type { EditCommand } from '../edit_history';
import type { EditorConfigPanel, EditorTarget, EditorToolContext } from '../editor_tool';
import { EditorTool } from '../editor_tool';
import type { MapDoc, MapDocBackgroundImage } from '../map_doc';
import type { BackgroundCorner } from './background_panel';
import { BackgroundPanel } from './background_panel';

class BackgroundCommand implements EditCommand {
  constructor(
    private before: MapDocBackgroundImage | undefined,
    private after: MapDocBackgroundImage | undefined
  ) {}

  apply(doc: MapDoc): void {
    doc.background_image = structuredClone(this.after);
  }

  revert(doc: MapDoc): void {
    doc.background_image = structuredClone(this.before);
  }
}

export class BackgroundTool extends EditorTool {
  readonly id = 'background';
  readonly label = 'Image';
  readonly hotkey = 'g';
  private panel: BackgroundPanel;

  constructor(context: EditorToolContext) {
    super(context);
    this.panel = new BackgroundPanel(context, {
      set: (name) => this.setImage(name),
      clear: () => this.commit(undefined),
      nudge: (corner, axis, delta) => this.nudge(corner, axis, delta),
    });
  }

  config(): EditorConfigPanel {
    return this.panel;
  }

  activated(): void {}
  deactivated(): void {}
  mousedown(_target: EditorTarget | undefined, _e: MouseEvent): boolean {
    return false;
  }
  mousemove(_target: EditorTarget | undefined): void {}
  mouseup(_target: EditorTarget | undefined, _e: MouseEvent): void {}

  drawOverlay(ctx: CanvasRenderingContext2D, _target: EditorTarget | undefined): void {
    const background = this.context.doc()?.background_image;
    if (!background) {
      return;
    }
    ctx.fillStyle = 'rgba(255, 220, 100, 0.25)';
    ctx.strokeStyle = 'rgba(255, 220, 100, 0.9)';
    ctx.lineWidth = 2;
    for (const [x, y] of [background.top_left, background.top_right]) {
      drawHexagon(ctx, this.context.toCanvas({ x, y }), this.context.hexR());
    }
  }

  private setImage(name: string): void {
    const doc = this.context.doc();
    if (!doc || !name) {
      return;
    }
    const size = doc.board_size;
    const current = doc.background_image;
    this.commit({
      name,
      top_left: current?.top_left ?? [-size, -size],
      top_right: current?.top_right ?? [size, -size],
    });
  }

  private nudge(corner: BackgroundCorner, axis: 0 | 1, delta: number): void {
    const next = structuredClone(this.context.doc()?.background_image);
    if (next) {
      next[corner][axis] += delta;
      this.commit(next);
    }
  }

  private commit(after: MapDocBackgroundImage | undefined): void {
    const doc = this.context.doc();
    if (!doc || JSON.stringify(doc.background_image) === JSON.stringify(after)) {
      return;
    }
    const command = new BackgroundCommand(doc.background_image, after);
    command.apply(doc);
    this.context.history.record(command);
    this.context.edited();
  }
}
