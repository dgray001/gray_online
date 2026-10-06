import { drawHexagon } from '../../game/util/canvas_util';
import type { Point2D } from '../../game/util/objects2d';
import { axialDistance } from '../../game/util/objects2d';
import type { EditCommand } from '../edit_history';
import type { EditorConfigPanel, EditorTarget, EditorToolContext } from '../editor_tool';
import { EditorTool } from '../editor_tool';
import type { MapDoc, MapDocSpace } from '../map_doc';
import { DEFAULT_TERRAIN_ID, findSpace } from '../map_doc';
import { TerrainPalette } from './terrain_palette';

interface TerrainChange {
  x: number;
  y: number;
  before: number;
  after: number;
}

class TerrainStroke implements EditCommand {
  private changes = new Map<string, TerrainChange>();

  record(x: number, y: number, before: number, after: number): void {
    const existing = this.changes.get(`${x},${y}`);
    if (existing) {
      existing.after = after;
    } else {
      this.changes.set(`${x},${y}`, { x, y, before, after });
    }
  }

  isEmpty(): boolean {
    return this.changes.size === 0;
  }

  apply(doc: MapDoc): void {
    this.setAll(doc, (change) => change.after);
  }

  revert(doc: MapDoc): void {
    this.setAll(doc, (change) => change.before);
  }

  private setAll(doc: MapDoc, terrain: (change: TerrainChange) => number): void {
    for (const change of this.changes.values()) {
      const space = findSpace(doc, change.x, change.y);
      if (space) {
        space.terrain = terrain(change);
      }
    }
  }
}

export class TerrainTool extends EditorTool {
  readonly id = 'terrain';
  readonly label = 'Terrain';
  readonly hotkey = 'b';
  private palette: TerrainPalette;
  private stroke?: TerrainStroke;

  constructor(context: EditorToolContext) {
    super(context);
    this.palette = new TerrainPalette(context);
  }

  config(): EditorConfigPanel {
    return this.palette;
  }

  activated(): void {}

  deactivated(): void {
    this.stroke = undefined;
  }

  mousedown(target: EditorTarget | undefined, e: MouseEvent): boolean {
    if (e.button !== 0 || !target) {
      return false;
    }
    this.stroke = new TerrainStroke();
    this.paint(target);
    return true;
  }

  mousemove(target: EditorTarget | undefined): void {
    if (this.stroke && target) {
      this.paint(target);
    }
  }

  mouseup(_target: EditorTarget | undefined, _e: MouseEvent): void {
    this.stroke = undefined;
  }

  drawOverlay(ctx: CanvasRenderingContext2D, target: EditorTarget | undefined): void {
    const doc = this.context.doc();
    if (!doc || !target) {
      return;
    }
    ctx.fillStyle = 'rgba(255, 255, 255, 0.12)';
    ctx.strokeStyle = 'rgba(255, 255, 255, 0.8)';
    ctx.lineWidth = 2;
    for (const space of this.brush(doc, target.space)) {
      drawHexagon(ctx, this.context.toCanvas(space), this.context.hexR());
    }
  }

  private brush(doc: MapDoc, center: Point2D): MapDocSpace[] {
    return doc.spaces.filter((space) => axialDistance(space, center) <= this.palette.brushRadius());
  }

  private paint(target: EditorTarget): void {
    const doc = this.context.doc();
    const stroke = this.stroke;
    if (!doc || !stroke) {
      return;
    }
    const terrain_id = this.palette.terrainId();
    const changed: Point2D[] = [];
    for (const space of this.brush(doc, target.space)) {
      const before = space.terrain ?? DEFAULT_TERRAIN_ID;
      if (before === terrain_id) {
        continue;
      }
      if (stroke.isEmpty()) {
        this.context.history.record(stroke);
      }
      stroke.record(space.x, space.y, before, terrain_id);
      space.terrain = terrain_id;
      changed.push({ x: space.x, y: space.y });
    }
    if (changed.length > 0) {
      this.context.patchTerrain(changed, terrain_id);
      this.context.edited();
    }
  }
}
