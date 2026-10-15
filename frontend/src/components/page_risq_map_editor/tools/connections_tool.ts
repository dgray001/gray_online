import { RisqTerrainType } from '../../game/games/risq/rendering/terrain';
import { drawHexagon, drawLine, drawText } from '../../game/util/canvas_util';
import type { Point2D } from '../../game/util/objects2d';
import type { EditorTarget, EditorToolContext } from '../editor_tool';
import { EditorTool } from '../editor_tool';
import type { MapDoc, MapDocConnection, MapDocSpace } from '../map_doc';
import { DEFAULT_TERRAIN_ID, arrivalDirection, findSpace } from '../map_doc';
import { ConnectionsPanel } from './connections_panel';

const DIRECTIONS: Point2D[] = [
  { x: 1, y: 0 },
  { x: 1, y: -1 },
  { x: 0, y: -1 },
  { x: -1, y: 0 },
  { x: -1, y: 1 },
  { x: 0, y: 1 },
];

export class ConnectionsTool extends EditorTool {
  readonly id = 'connections';
  readonly label = 'Links';
  readonly hotkey = 'c';
  private direction = 0;
  private to_direction = 3;
  private pending?: MapDocSpace;
  private message = 'Choose the first space.';
  private panel: ConnectionsPanel;

  constructor(context: EditorToolContext) {
    super(context);
    this.panel = new ConnectionsPanel(
      () => (this.pending ? `Arrive at edge ${this.to_direction}` : `Leave by edge ${this.direction}`),
      () => {
        this.syncPending();
        return this.message;
      },
      () => {
        if (this.pending) {
          this.to_direction = (this.to_direction + 1) % 6;
        } else {
          this.direction = (this.direction + 1) % 6;
          this.to_direction = (this.direction + 3) % 6;
        }
      }
    );
  }
  config(): ConnectionsPanel {
    return this.panel;
  }
  activated(): void {
    this.deactivated();
  }
  deactivated(): void {
    this.pending = undefined;
    this.to_direction = (this.direction + 3) % 6;
    this.message = 'Choose the first space.';
  }
  private syncPending(): void {
    const doc = this.context.doc();
    if (this.pending && (!doc || findSpace(doc, this.pending.x, this.pending.y) !== this.pending)) {
      this.deactivated();
    }
  }
  mousemove(_target: EditorTarget | undefined): void {}
  mousedown(target: EditorTarget | undefined, e: MouseEvent): boolean {
    const doc = this.context.doc();
    if (e.button !== 0 || !doc || !target) {
      return false;
    }
    this.syncPending();
    const space = findSpace(doc, target.space.x, target.space.y);
    if (!space) {
      return true;
    }
    const terrain = this.context.terrains().find((entry) => entry.terrain_id === (space.terrain ?? DEFAULT_TERRAIN_ID));
    if (terrain?.terrain_type === RisqTerrainType.WATER || terrain?.terrain_type === RisqTerrainType.DEEP_WATER) {
      this.message = 'Choose a passable space.';
    } else if (this.pending === space) {
      this.deactivated();
    } else if (!this.pending) {
      this.pending = space;
      this.message = `From (${space.x}, ${space.y}); pick arrival edge, choose destination.`;
    } else {
      this.toggleConnection(doc, this.pending, space);
    }
    return true;
  }
  mouseup(_target: EditorTarget | undefined, _e: MouseEvent): void {}
  private toggleConnection(doc: MapDoc, from: Point2D, to: Point2D): void {
    const connections = doc.connections ?? [];
    const index = connections.findIndex(
      (link) =>
        (link.from[0] === from.x &&
          link.from[1] === from.y &&
          link.to[0] === to.x &&
          link.to[1] === to.y &&
          link.direction === this.direction &&
          arrivalDirection(link) === this.to_direction) ||
        (link.from[0] === to.x &&
          link.from[1] === to.y &&
          link.to[0] === from.x &&
          link.to[1] === from.y &&
          link.direction === this.to_direction &&
          arrivalDirection(link) === this.direction)
    );
    if (index < 0 && (this.edgeOccupied(doc, from, this.direction) || this.edgeOccupied(doc, to, this.to_direction))) {
      this.message = 'That edge is already connected.';
      return;
    }
    const after = structuredClone(connections);
    if (index >= 0) {
      after.splice(index, 1);
    } else {
      const link: MapDocConnection = { from: [from.x, from.y], to: [to.x, to.y], direction: this.direction };
      if (this.to_direction !== (this.direction + 3) % 6) {
        link.to_direction = this.to_direction;
      }
      after.push(link);
    }
    this.commit(doc, after);
    this.deactivated();
    this.message = index >= 0 ? 'Connection removed.' : 'Connection added.';
  }
  private commit(doc: MapDoc, after: MapDocConnection[]): void {
    const before = structuredClone(doc.connections);
    this.context.history.record({
      apply: (target: MapDoc): void => {
        target.connections = structuredClone(after);
      },
      revert: (target: MapDoc): void => {
        target.connections = structuredClone(before);
      },
    });
    doc.connections = after;
    this.context.edited();
  }
  private edgeOccupied(doc: MapDoc, space: Point2D, direction: number): boolean {
    const vector = DIRECTIONS[direction];
    return (
      !!findSpace(doc, space.x + vector.x, space.y + vector.y) ||
      (doc.connections ?? []).some(
        (link) =>
          (link.from[0] === space.x && link.from[1] === space.y && link.direction === direction) ||
          (link.to[0] === space.x && link.to[1] === space.y && arrivalDirection(link) === direction)
      )
    );
  }
  private drawConnection(
    ctx: CanvasRenderingContext2D,
    from: Point2D,
    to: Point2D,
    direction: number,
    to_direction: number
  ): void {
    ctx.save();
    const start = this.connectionPoint(from, direction);
    const end = this.connectionPoint(to, to_direction);
    drawLine(ctx, start, end);
    for (const point of [start, end]) {
      drawHexagon(ctx, point, this.context.hexR() * 0.3);
    }
    drawText(ctx, `${direction} > ${to_direction}`, {
      p: { x: (start.x + end.x) / 2, y: (start.y + end.y) / 2 },
      w: 90,
      fill_style: ctx.strokeStyle as string,
      font: '12px "Fira Sans", sans-serif',
      align: 'center',
    });
    ctx.restore();
  }
  private connectionPoint(space: Point2D, direction: number): Point2D {
    const center = this.context.toCanvas(space);
    const vector = DIRECTIONS[direction];
    const neighbor = this.context.toCanvas({ x: space.x + vector.x, y: space.y + vector.y });
    return { x: center.x + (neighbor.x - center.x) * 0.4, y: center.y + (neighbor.y - center.y) * 0.4 };
  }
  drawOverlay(ctx: CanvasRenderingContext2D, target: EditorTarget | undefined): void {
    this.syncPending();
    ctx.save();
    ctx.fillStyle = 'transparent';
    ctx.strokeStyle = 'rgb(90, 220, 255)';
    ctx.lineWidth = 2;
    for (const link of this.context.doc()?.connections ?? []) {
      this.drawConnection(
        ctx,
        { x: link.from[0], y: link.from[1] },
        { x: link.to[0], y: link.to[1] },
        link.direction,
        arrivalDirection(link)
      );
    }
    if (this.pending) {
      ctx.strokeStyle = 'rgb(255, 220, 80)';
      ctx.lineWidth = 3;
      drawHexagon(ctx, this.context.toCanvas(this.pending), this.context.hexR());
      if (target) {
        ctx.setLineDash([8, 5]);
        this.drawConnection(ctx, this.pending, target.space, this.direction, this.to_direction);
      }
    }
    ctx.restore();
  }
}
