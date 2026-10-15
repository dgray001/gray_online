import type { EditCommand } from '../edit_history';
import type { EditorConfigPanel, EditorTarget, EditorToolContext } from '../editor_tool';
import { EditorTool } from '../editor_tool';
import type { MapDoc, MapDocBank, MapDocSpace } from '../map_doc';
import { findSpace } from '../map_doc';
import { SlotsPanel } from './slots_panel';

type SlotFields = Pick<MapDocSpace, 'zones' | 'player_slot'>;

interface SlotsSnapshot {
  players: number;
  starting_bank?: MapDocBank;
}

interface SpaceChange {
  x: number;
  y: number;
  before: SlotFields;
  after: SlotFields;
}

class SlotsEditCommand implements EditCommand {
  constructor(
    private before: SlotsSnapshot,
    private after: SlotsSnapshot,
    private spaces: SpaceChange[]
  ) {}

  apply(doc: MapDoc): void {
    this.restore(doc, this.after, (change) => change.after);
  }

  revert(doc: MapDoc): void {
    this.restore(doc, this.before, (change) => change.before);
  }

  private restore(doc: MapDoc, snapshot: SlotsSnapshot, fields: (change: SpaceChange) => SlotFields): void {
    doc.players = snapshot.players;
    doc.starting_bank = structuredClone(snapshot.starting_bank);
    for (const change of this.spaces) {
      const space = findSpace(doc, change.x, change.y);
      if (space) {
        const { zones, player_slot } = structuredClone(fields(change));
        Object.assign(space, { zones, player_slot });
      }
    }
  }
}
export class SlotsTool extends EditorTool {
  readonly id = 'slots';
  readonly label = 'Slots';
  readonly hotkey = 'l';
  private panel: SlotsPanel;

  constructor(context: EditorToolContext) {
    super(context);
    this.panel = new SlotsPanel(
      context,
      (delta) => this.changeSlots(delta),
      (res, delta) => this.changeBank(res, delta)
    );
  }

  config(): EditorConfigPanel {
    return this.panel;
  }

  activated(): void {}
  deactivated(): void {}
  private snapshot(doc: MapDoc): SlotsSnapshot {
    return {
      players: doc.players,
      starting_bank: structuredClone(doc.starting_bank),
    };
  }

  private changeSlots(delta: number): void {
    const doc = this.context.doc();
    if (!doc) {
      return;
    }
    const next_players = Math.max(1, Math.min(12, doc.players + delta));
    if (next_players === doc.players) {
      return;
    }
    const before = this.snapshot(doc);
    doc.players = next_players;
    const changes = delta < 0 ? this.stripPlayerEntities(doc, next_players) : [];
    this.context.history.record(new SlotsEditCommand(before, this.snapshot(doc), changes));
    this.context.edited();
  }
  private changeBank(resource: 'food' | 'wood' | 'stone' | 'gold', delta: number): void {
    const doc = this.context.doc();
    if (!doc) {
      return;
    }
    const current = doc.starting_bank?.[resource] ?? 0;
    if (typeof current !== 'number') {
      return; // an expression over the setup settings; edit it in the map file
    }
    const before = this.snapshot(doc);
    doc.starting_bank = doc.starting_bank ?? { food: 0, wood: 0, stone: 0, gold: 0 };
    doc.starting_bank[resource] = Math.max(0, current + delta);
    if (JSON.stringify(before.starting_bank) === JSON.stringify(doc.starting_bank)) {
      doc.starting_bank = before.starting_bank;
      return;
    }
    this.context.history.record(new SlotsEditCommand(before, this.snapshot(doc), []));
    this.context.edited();
  }

  private stripPlayerEntities(doc: MapDoc, max_players: number): SpaceChange[] {
    const changes: SpaceChange[] = [];
    for (const space of doc.spaces) {
      const before = structuredClone({ zones: space.zones, player_slot: space.player_slot });
      this.stripSpace(space, max_players);
      const after = structuredClone({ zones: space.zones, player_slot: space.player_slot });
      if (JSON.stringify(before) !== JSON.stringify(after)) {
        changes.push({ x: space.x, y: space.y, before, after });
      }
    }
    return changes;
  }

  private stripSpace(space: MapDocSpace, max_players: number): void {
    if (space.player_slot !== undefined && space.player_slot >= max_players) {
      delete space.player_slot;
    }
    for (const zone of space.zones ?? []) {
      if (zone.building && zone.building.player >= max_players) {
        delete zone.building;
      }
      if (zone.units) {
        zone.units = zone.units.filter((u) => u.player < max_players);
        if (zone.units.length === 0) {
          delete zone.units;
        }
      }
    }
    if (space.zones) {
      space.zones = space.zones.filter((z) => z.resource || z.building || z.units?.length || z.terrain_override);
    }
  }
  mousedown(_target: EditorTarget | undefined, _e: MouseEvent): boolean {
    return false;
  }
  mousemove(_target: EditorTarget | undefined): void {}
  mouseup(_target: EditorTarget | undefined, _e: MouseEvent): void {}

  drawOverlay(ctx: CanvasRenderingContext2D, target: EditorTarget | undefined): void {
    if (!target) {
      return;
    }
    const canvas_pt = this.context.toCanvas(target.space);
    ctx.strokeStyle = 'rgba(255, 255, 255, 0.4)';
    ctx.lineWidth = 1.5;
    ctx.strokeRect(canvas_pt.x - 20, canvas_pt.y - 20, 40, 40);
  }
}
