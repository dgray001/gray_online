import { clipToZone } from '../../game/games/risq/rendering/zones/geometry';
import type { Point2D } from '../../game/util/objects2d';
import type { EditCommand } from '../edit_history';
import type { EditorConfigPanel, EditorTarget, EditorToolContext } from '../editor_tool';
import { EditorTool } from '../editor_tool';
import type { MapDoc, MapDocSpace, MapDocZone } from '../map_doc';
import { findSpace } from '../map_doc';
import { ZonePalette } from './zone_palette';

interface ZoneSnapshot {
  space: Point2D;
  zone: Point2D;
  before?: MapDocZone;
  after?: MapDocZone;
}
class ZoneEditCommand implements EditCommand {
  private changes: ZoneSnapshot[] = [];

  record(change: ZoneSnapshot): void {
    this.changes.push(change);
  }

  isEmpty(): boolean {
    return this.changes.length === 0;
  }

  apply(doc: MapDoc): void {
    this.changes.forEach((c) => this.setZone(doc, c.space, c.zone, c.after));
  }

  revert(doc: MapDoc): void {
    for (let i = this.changes.length - 1; i >= 0; i--) {
      const c = this.changes[i];
      this.setZone(doc, c.space, c.zone, c.before);
    }
  }
  private setZone(doc: MapDoc, space_pos: Point2D, zone_pos: Point2D, zone_doc?: MapDocZone): void {
    const space = findSpace(doc, space_pos.x, space_pos.y);
    if (!space) {
      return;
    }
    space.zones = (space.zones ?? []).filter((z) => !(z.x === zone_pos.x && z.y === zone_pos.y));
    if (zone_doc) {
      space.zones.push(JSON.parse(JSON.stringify(zone_doc)));
    }
  }
}
export class ZoneTool extends EditorTool {
  readonly id = 'zone';
  readonly label = 'Zones';
  readonly hotkey = 'z';
  private palette: ZonePalette;

  constructor(context: EditorToolContext) {
    super(context);
    this.palette = new ZonePalette(context);
  }

  config(): EditorConfigPanel {
    return this.palette;
  }

  activated(): void {}
  deactivated(): void {}
  mousedown(target: EditorTarget | undefined, e: MouseEvent): boolean {
    if (e.button !== 0 || !target) {
      return false;
    }
    const zone_coord = target.zone ?? { x: 0, y: 0 };
    this.mutateZone(target.space, zone_coord);
    return true;
  }

  mousemove(_target: EditorTarget | undefined): void {}
  mouseup(_target: EditorTarget | undefined, _e: MouseEvent): void {}
  drawOverlay(ctx: CanvasRenderingContext2D, target: EditorTarget | undefined): void {
    const doc = this.context.doc();
    if (!doc || !target) {
      return;
    }
    const zone_coord = target.zone ?? { x: 0, y: 0 };
    const canvas_pt = this.context.toCanvas(target.space);
    const hex_r = this.context.hexR();
    ctx.save();
    clipToZone(ctx, canvas_pt, hex_r, zone_coord);
    ctx.fillStyle = 'rgba(255, 220, 100, 0.25)';
    ctx.fillRect(canvas_pt.x - hex_r, canvas_pt.y - hex_r, 2 * hex_r, 2 * hex_r);
    ctx.strokeStyle = 'rgba(255, 220, 100, 0.9)';
    ctx.lineWidth = 2;
    ctx.stroke();
    ctx.restore();
  }
  private mutateZone(space_pos: Point2D, zone_pos: Point2D): void {
    const doc = this.context.doc();
    if (!doc) {
      return;
    }
    const space = findSpace(doc, space_pos.x, space_pos.y);
    if (!space) {
      return;
    }
    space.zones = space.zones ?? [];
    let zone = space.zones.find((z) => z.x === zone_pos.x && z.y === zone_pos.y);
    const before: MapDocZone | undefined = zone ? JSON.parse(JSON.stringify(zone)) : undefined;
    if (!zone) {
      zone = { x: zone_pos.x, y: zone_pos.y };
      space.zones.push(zone);
    }
    this.applyPaletteToZone(zone);
    this.cleanEmptyZone(space, zone);
    const after: MapDocZone | undefined = space.zones.find((z) => z.x === zone_pos.x && z.y === zone_pos.y);
    const command = new ZoneEditCommand();
    command.record({
      space: space_pos,
      zone: zone_pos,
      before,
      after: after ? JSON.parse(JSON.stringify(after)) : undefined,
    });
    this.context.history.record(command);
    this.context.edited();
  }
  private applyPaletteToZone(zone: MapDocZone): void {
    const mode = this.palette.mode();
    if (mode === 'resource') {
      delete zone.building;
      zone.resource = this.palette.resourceId();
    } else if (mode === 'building') {
      delete zone.resource;
      zone.building = { id: this.palette.buildingId(), player: this.palette.playerSlot() };
    } else if (mode === 'unit') {
      zone.units = [{ id: this.palette.unitId(), player: this.palette.playerSlot(), count: this.palette.unitCount() }];
    } else if (mode === 'terrain') {
      zone.terrain_override = this.palette.terrainId();
    } else if (mode === 'clear') {
      delete zone.resource;
      delete zone.building;
      delete zone.units;
      delete zone.terrain_override;
    }
  }

  private cleanEmptyZone(space: MapDocSpace, zone: MapDocZone): void {
    if (!zone.resource && !zone.building && (!zone.units || zone.units.length === 0) && !zone.terrain_override) {
      space.zones = (space.zones ?? []).filter((z) => z !== zone);
    }
  }
}
