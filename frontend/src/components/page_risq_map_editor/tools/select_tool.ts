import type { EditorConfigPanel, EditorTarget } from '../editor_tool';
import { EditorTool } from '../editor_tool';

export class SelectTool extends EditorTool {
  readonly id = 'select';
  readonly label = 'Select';
  readonly hotkey = 'v';

  config(): EditorConfigPanel | undefined {
    return undefined;
  }

  activated(): void {}

  deactivated(): void {
    this.context.select(undefined);
  }

  mousedown(_target: EditorTarget | undefined, _e: MouseEvent): boolean {
    return false;
  }

  mousemove(_target: EditorTarget | undefined): void {}

  mouseup(target: EditorTarget | undefined, e: MouseEvent): void {
    if (e.button === 0 && !this.context.dragged()) {
      this.context.select(target);
    }
  }

  drawOverlay(_ctx: CanvasRenderingContext2D, _target: EditorTarget | undefined): void {}
}
