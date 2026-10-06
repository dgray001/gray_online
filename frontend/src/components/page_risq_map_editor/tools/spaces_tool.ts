import { drawHexagon } from '../../game/util/canvas_util';
import type { Point2D } from '../../game/util/objects2d';
import { axialDistance } from '../../game/util/objects2d';
import type { EditCommand } from '../edit_history';
import type { EditorConfigPanel, EditorTarget, EditorToolContext } from '../editor_tool';
import { EditorTool } from '../editor_tool';
import type { MapDoc, MapDocSpace } from '../map_doc';
import { DEFAULT_TERRAIN_ID, findSpace } from '../map_doc';
class SpaceCommand implements EditCommand {
  private added = new Map<string, MapDocSpace>();
  private erased = new Map<string, MapDocSpace>();
  private references_before?: Pick<MapDoc, 'regions' | 'connections'>;
  private references_after?: Pick<MapDoc, 'regions' | 'connections'>;

  eraseReferences(doc: MapDoc, coord: Point2D): void {
    this.references_before ??= structuredClone({ regions: doc.regions, connections: doc.connections });
    for (const region of doc.regions ?? []) {
      region.spaces = region.spaces.filter(([x, y]) => x !== coord.x || y !== coord.y);
    }
    const key = `${coord.x},${coord.y}`;
    doc.connections = doc.connections?.filter(
      (connection) => connection.from.join(',') !== key && connection.to.join(',') !== key
    );
    this.references_after = structuredClone({ regions: doc.regions, connections: doc.connections });
  }

  recordAdd(space: MapDocSpace): void {
    const key = `${space.x},${space.y}`;
    if (this.erased.has(key)) {
      this.erased.delete(key);
    } else {
      this.added.set(key, space);
    }
  }

  recordErase(space: MapDocSpace): void {
    const key = `${space.x},${space.y}`;
    if (this.added.has(key)) {
      this.added.delete(key);
    } else {
      this.erased.set(key, space);
    }
  }
  isEmpty(): boolean {
    return this.added.size === 0 && this.erased.size === 0;
  }

  apply(doc: MapDoc): void {
    const erase_keys = new Set(this.erased.keys());
    doc.spaces = doc.spaces.filter((s) => !erase_keys.has(`${s.x},${s.y}`));
    for (const space of this.added.values()) {
      doc.spaces.push(space);
    }
    if (this.references_after) {
      Object.assign(doc, structuredClone(this.references_after));
    }
  }

  revert(doc: MapDoc): void {
    const add_keys = new Set(this.added.keys());
    doc.spaces = doc.spaces.filter((s) => !add_keys.has(`${s.x},${s.y}`));
    for (const space of this.erased.values()) {
      doc.spaces.push(space);
    }
    if (this.references_before) {
      Object.assign(doc, structuredClone(this.references_before));
    }
  }
}
export class SpacesTool extends EditorTool {
  readonly id = 'spaces';
  readonly label = 'Spaces';
  readonly hotkey = 's';
  private command?: SpaceCommand;
  private mode?: 'add' | 'erase';

  constructor(context: EditorToolContext) {
    super(context);
  }

  config(): EditorConfigPanel | undefined {
    return undefined;
  }

  activated(): void {}

  deactivated(): void {
    this.command = undefined;
    this.mode = undefined;
  }
  mousedown(target: EditorTarget | undefined, e: MouseEvent): boolean {
    if (e.button !== 0 || !target) {
      return false;
    }
    const doc = this.context.doc();
    if (!doc) {
      return false;
    }
    this.command = new SpaceCommand();
    this.mode = findSpace(doc, target.space.x, target.space.y) ? 'erase' : 'add';
    this.mutate(target.space);
    return true;
  }

  mousemove(target: EditorTarget | undefined): void {
    if (this.command && this.mode && target) {
      this.mutate(target.space);
    }
  }

  mouseup(_target: EditorTarget | undefined, _e: MouseEvent): void {
    this.command = undefined;
    this.mode = undefined;
  }
  drawOverlay(ctx: CanvasRenderingContext2D, target: EditorTarget | undefined): void {
    const doc = this.context.doc();
    if (!doc) {
      return;
    }
    const hex_r = this.context.hexR();
    ctx.strokeStyle = 'rgba(255, 255, 255, 0.12)';
    ctx.lineWidth = 1;
    for (let x = -doc.board_size; x <= doc.board_size; x++) {
      for (
        let y = Math.max(-doc.board_size, -x - doc.board_size);
        y <= Math.min(doc.board_size, -x + doc.board_size);
        y++
      ) {
        if (!findSpace(doc, x, y)) {
          drawHexagon(ctx, this.context.toCanvas({ x, y }), hex_r);
        }
      }
    }
    if (target && axialDistance(target.space, { x: 0, y: 0 }) <= doc.board_size) {
      const exists = !!findSpace(doc, target.space.x, target.space.y);
      ctx.strokeStyle = exists ? 'rgba(255, 70, 70, 0.9)' : 'rgba(70, 255, 70, 0.9)';
      ctx.fillStyle = exists ? 'rgba(255, 70, 70, 0.2)' : 'rgba(70, 255, 70, 0.2)';
      ctx.lineWidth = 2;
      drawHexagon(ctx, this.context.toCanvas(target.space), hex_r);
    }
  }
  private mutate(coord: Point2D): void {
    const doc = this.context.doc();
    const command = this.command;
    if (!doc || !command || axialDistance(coord, { x: 0, y: 0 }) > doc.board_size) {
      return;
    }
    const existing = findSpace(doc, coord.x, coord.y);
    if (this.mode === 'add' && !existing) {
      const space: MapDocSpace = { x: coord.x, y: coord.y, terrain: DEFAULT_TERRAIN_ID };
      if (command.isEmpty()) {
        this.context.history.record(command);
      }
      command.recordAdd(space);
      doc.spaces.push(space);
      this.context.edited();
    } else if (this.mode === 'erase' && existing) {
      if (
        doc.player_start &&
        doc.spaces.some(
          (space) => space.player_slot !== undefined && axialDistance(space, coord) <= doc.player_start!.size
        )
      ) {
        return;
      }
      if (command.isEmpty()) {
        this.context.history.record(command);
      }
      command.recordErase(existing);
      doc.spaces = doc.spaces.filter((s) => s !== existing);
      command.eraseReferences(doc, coord);
      this.context.edited();
    }
  }
}
