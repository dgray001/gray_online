import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { DwgRectButton } from '../../../../util/canvas_components/button/rect_button';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawRect, drawText } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import { unitImage } from '../../rendering/assets/unit';
import { buildingImage } from '../../rendering/assets/buildings';
import { controlGroupTiles } from './control_group_tiles';
import { RisqControlGroupIcon } from './control_group_icon';
import { RisqControlGroupCloseButton } from './control_group_close_button';
import { controlGroupLayout, CONTROL_ICON_SIZE, CONTROL_GAP, CONTROL_LABEL } from './control_group_layout';
import type { ControlGroupBoxLayout } from './control_group_layout';
import type { ControlGroup } from '../../application/selection/control_groups';
import { clipBottomPanelItem } from './clip_item';

export class RisqControlGroupGrid extends DwgRectButton {
  private buttons: RisqControlGroupIcon[] = [];
  private close_buttons: RisqControlGroupCloseButton[] = [];
  private controls: DwgRectButton[] = [];
  private boxes: (ControlGroupBoxLayout & { number: number })[] = [];
  private layout_key = '';
  private snapshot: ReturnType<DwgRisq['getGame']>;
  private available_width = 0;
  private content_width = 0;
  private scroll_x = 0;
  constructor(private risq: DwgRisq) {
    super({
      button_config: {},
      p: { x: 0, y: 0 },
      w: 0,
      h: 0,
      draw_config: { fill_style: 'transparent', stroke_width: 0, fixed_position: true },
    });
  }
  setAvailableWidth(width: number): void {
    this.available_width = Math.max(0, width);
    this.dataRefreshed();
  }
  override setPosition(p: Point2D): void {
    if (p.x === this.xi() && p.y === this.yi()) {
      return;
    }
    super.setPosition(p);
    this.layout_key = '';
    this.dataRefreshed();
  }
  dataRefreshed(): void {
    const key = JSON.stringify([this.available_width, this.xi(), this.yi(), this.risq.control_groups.revision]);
    const game = this.risq.getGame();
    if (key === this.layout_key && game === this.snapshot) {
      return;
    }
    this.layout_key = key;
    this.snapshot = game;
    const entries = this.risq.control_groups.entries();
    this.buttons = [];
    this.close_buttons = [];
    this.controls = [];
    this.boxes = controlGroupLayout(
      entries.map(([, group]) => group.ids.length),
      this.available_width
    ).map((box, i) => ({ ...box, number: entries[i][0] }));
    this.rebuildIcons(entries);
  }
  private rebuildIcons(entries: [number, ControlGroup][]): void {
    const player = this.risq.getPlayer();
    if (!player) {
      this.setSize(0, 0);
      return;
    }
    const color = this.risq.getGame()!.players[player.player.player_id].color;
    this.content_width = 0;
    let row_x = 0;
    let row_y = -1;
    for (const [i, [, group]] of entries.entries()) {
      const box = this.boxes[i];
      if (box.y !== row_y) {
        row_x = 0;
        row_y = box.y;
      }
      const building = group.kind === 'building' ? player.buildings.get(group.ids[0]) : undefined;
      const tiles = building
        ? group.ids.map((id: number): { image: string; count: number; ids: number[] } => {
            const member = player.buildings.get(id)!;
            return { image: buildingImage(member.building_id, member.under_construction), count: 1, ids: [id] };
          })
        : controlGroupTiles(player.units, group.ids, box.cols * box.rows).map(({ unit, count, ids }) => ({
            image: unitImage(unit.unit_id),
            count,
            ids,
          }));
      box.cols = Math.max(1, Math.ceil(tiles.length / box.rows));
      box.width = box.cols * (CONTROL_ICON_SIZE + CONTROL_GAP) + CONTROL_GAP;
      box.x = row_x;
      row_x += box.width + CONTROL_GAP;
      this.content_width = Math.max(this.content_width, row_x - CONTROL_GAP);
      this.close_buttons.push(
        new RisqControlGroupCloseButton(
          {
            x: this.xi() + box.x + box.width - CONTROL_LABEL + 1 - this.scroll_x,
            y: this.yi() + box.y + 1,
          },
          CONTROL_LABEL - 2,
          (): void => {
            this.risq.control_groups.removeGroup(box.number);
            this.dataRefreshed();
          }
        )
      );
      for (const [j, tile] of tiles.entries()) {
        this.buttons.push(
          new RisqControlGroupIcon(
            this.risq.getPlayerColoredIcon(tile.image, color),
            {
              x: this.xi() + box.x + CONTROL_GAP + (j % box.cols) * (CONTROL_ICON_SIZE + CONTROL_GAP) - this.scroll_x,
              y: this.yi() + box.y + CONTROL_LABEL + Math.floor(j / box.cols) * (CONTROL_ICON_SIZE + CONTROL_GAP),
            },
            CONTROL_ICON_SIZE,
            tile.count,
            { kind: group.kind, ids: tile.ids },
            (event: MouseEvent): void => this.activateIcon(entries[i][0], { kind: group.kind, ids: tile.ids }, event)
          )
        );
      }
    }
    this.setSize(
      Math.min(this.available_width, this.content_width),
      entries.length
        ? Math.max(...this.boxes.map((box) => box.y + CONTROL_LABEL + box.rows * (CONTROL_ICON_SIZE + CONTROL_GAP)))
        : 0
    );
    const clamped_scroll = Math.min(this.scroll_x, Math.max(0, this.content_width - this.w()));
    this.controls = [...this.buttons, ...this.close_buttons];
    if (clamped_scroll !== this.scroll_x) {
      for (const button of this.controls) {
        button.setPosition({ x: button.xi() + this.scroll_x - clamped_scroll, y: button.yi() });
      }
      this.scroll_x = clamped_scroll;
    }
  }
  private activateIcon(number: number, subject: ControlGroup, e: MouseEvent): void {
    const id = subject.ids[0];
    const player = this.risq.getPlayer();
    const unit = subject.kind === 'unit' ? player?.units.get(id) : undefined;
    if (e.button === 2) {
      const ids = e.shiftKey && unit ? this.risq.control_groups.unitIds(number, unit.unit_id) : [id];
      this.risq.control_groups.removeMembers(number, ids);
      this.dataRefreshed();
      return;
    }
    if (subject.kind === 'building') {
      const building = player?.buildings.get(id);
      if (building) {
        const ids = e.shiftKey
          ? (
              this.risq.control_groups.entries().find(([n]: [number, ControlGroup]): boolean => n === number)?.[1]
                .ids ?? []
            ).filter(
              (candidate: number): boolean => player?.buildings.get(candidate)?.building_id === building.building_id
            )
          : [id];
        const selected = new Set(this.risq.selection.selectedBuildingIds());
        if (e.ctrlKey) {
          const remove = ids.every((candidate: number): boolean => selected.has(candidate));
          for (const candidate of ids) {
            if (remove) {
              selected.delete(candidate);
            } else {
              selected.add(candidate);
            }
          }
        }
        this.risq.selection.selectOwnBuildings(e.ctrlKey ? [...selected] : ids);
      }
      return;
    }
    if (!unit) {
      return;
    }
    const ids = e.shiftKey ? this.risq.control_groups.unitIds(number, unit.unit_id) : [id];
    if (e.ctrlKey) {
      const selected = this.risq.selection.selectedUnitIds();
      if (selected.has(id)) {
        ids.forEach((internal_id) => selected.delete(internal_id));
      } else {
        ids.forEach((internal_id) => selected.add(internal_id));
      }
      this.risq.selection.selectOwnUnits([...selected]);
    } else if (e.shiftKey) {
      this.risq.selection.selectOwnUnits(ids);
    } else {
      this.risq.selection.selectUnit(unit);
    }
  }

  protected override _draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    ctx.save();
    clipBottomPanelItem(ctx, transform, this);
    this.drawBoxes(ctx, transform);
    const selected_units = this.risq.selection.selectedUnitIds();
    const selected_buildings = new Set(this.risq.selection.selectedBuildingIds());
    for (const button of this.buttons) {
      button.setSelection(selected_units, selected_buildings);
      button.draw(ctx, transform, dt);
    }
    for (const button of this.close_buttons) {
      button.draw(ctx, transform, dt);
    }
    ctx.restore();
  }
  private drawBoxes(ctx: CanvasRenderingContext2D, transform: BoardTransformData): void {
    configDraw(
      ctx,
      transform,
      {
        fill_style: 'transparent',
        stroke_style: 'black',
        stroke_width: 1,
        fixed_position: true,
      },
      false,
      false,
      () => {
        for (const box of this.boxes) {
          const p = { x: this.xi() + box.x - this.scroll_x, y: this.yi() + box.y };
          ctx.fillStyle = 'transparent';
          ctx.strokeStyle = 'black';
          drawRect(ctx, p, box.width, CONTROL_LABEL + box.rows * (CONTROL_ICON_SIZE + CONTROL_GAP));
          drawText(ctx, `${box.number % 10}`, {
            p: { x: p.x + CONTROL_GAP - 1, y: p.y + 3 },
            w: box.width - 2 * CONTROL_GAP,
            fill_style: 'black',
            align: 'left',
            baseline: 'top',
            font: 'bold 12px serif',
          });
        }
      }
    );
  }
  override mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    const hovering = super.mousemove(canvas, screen, transform);
    for (const button of this.controls) {
      if (hovering) {
        button.mousemove(canvas, screen, transform);
      } else {
        button.setHovering(false);
      }
    }
    return hovering;
  }
  override mousedown(e: MouseEvent): boolean {
    return this.controls.map((button) => button.mousedown(e)).some(Boolean);
  }
  override mouseup(e: MouseEvent): void {
    this.controls.forEach((button) => button.mouseup(e));
  }
  scroll(dy: number, mode: number): boolean {
    const next = Math.max(0, Math.min(this.content_width - this.w(), this.scroll_x + dy * (mode ? 16 : 1)));
    if (next === this.scroll_x) {
      return false;
    }
    this.scroll_x = next;
    this.layout_key = '';
    this.dataRefreshed();
    return true;
  }
  override setHovering(hovering: boolean): void {
    super.setHovering(hovering);
    if (!hovering) {
      this.controls.forEach((button) => button.setHovering(false));
    }
  }
  override setClicking(clicking: boolean): void {
    super.setClicking(clicking);
    if (!clicking) {
      this.controls.forEach((button) => button.setClicking(false));
    }
  }
  protected hovered(): void {}
  protected unhovered(): void {}
  protected clicked(): void {}
  protected released(): void {}
}
