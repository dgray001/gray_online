import type { BoardTransformData } from '../../../../../util/canvas_board/canvas_board';
import { drawRect, drawText } from '../../../../../util/canvas_util';
import type { Point2D } from '../../../../../util/objects2d';
import { hasBuildings, hasEconomy } from '../../../application/session';
import { capitalize } from '../../../../../../../scripts/util';
import type { RectHoverData, RisqBuilding, RisqRegion, RisqResource, RisqSpace } from '../../../model/types';
import { RisqResourceType, RisqVisibilityLevel } from '../../../model/types';
import { resourceType, resourceTypeImage } from '../../../rendering/assets/resources';
import type { DwgRisq } from '../../../risq';
import type { RisqTooltipData } from '../../risq_tooltip';
import { RisqButtonTooltip } from '../../risq_tooltip';
import { COMBO_UNIT_ICON_SIZE, comboUnitIconKey, drawComboUnitIcon } from '../../../rendering/assets/unit';
import { spaceOwnerColor, terrainTypeLabel } from '../../../rendering/terrain';
import { RisqSpaceUnitsRowButton } from '../controls/space_units_row_button';
import { invertPair } from '../../../model/coordinates';
import type { PanelDrawContext } from './primitives';
import { drawName, drawSeparator, drawSubtitle } from './primitives';
import type { RisqSpaceHexagon } from './space_hexagon';
import { militaryTooltipData, villagerTooltipData } from './space_unit_tooltips';

/** Draws a combo villager+military icon (just the military icon without an economy), used where only a total unit count is visible */
export function comboUnitIcon(pc: PanelDrawContext): CanvasImageSource | undefined {
  const villager_img = pc.risq.getIcon('icons/villager64');
  const unit_img = pc.risq.getIcon('icons/unit64');
  if (!hasEconomy(pc.risq.getGame())) {
    return unit_img;
  }
  return pc.risq
    .getImageCache()
    .getImage(comboUnitIconKey(false), COMBO_UNIT_ICON_SIZE, [villager_img, unit_img], (combo_ctx) =>
      drawComboUnitIcon(combo_ctx, villager_img, unit_img)
    );
}

declare interface SpaceResourceRow {
  type: RisqResourceType;
  total: number;
  count: number;
  breakdown: string[];
}

function resourceBreakdown(resources: RisqResource[]): string[] {
  const groups = new Map<string, { count: number; total: number }>();
  for (const resource of resources) {
    const group = groups.get(resource.display_name) ?? { count: 0, total: 0 };
    groups.set(resource.display_name, { count: group.count + 1, total: group.total + resource.resources_left });
  }
  return [...groups].map(([name, group]) => `${group.count} ${name}: ${Math.round(group.total)}`);
}

function resourceRows(space: RisqSpace): SpaceResourceRow[] {
  const by_type = new Map<RisqResourceType, RisqResource[]>();
  for (const resource of [...(space.resources?.values() ?? [])].filter((r) => r.resources_left > 0)) {
    by_type.set(resourceType(resource), [...(by_type.get(resourceType(resource)) ?? []), resource]);
  }
  return [...by_type]
    .sort(([a], [b]) => a - b)
    .map(([type, resources]) => ({
      type,
      total: Math.round(resources.reduce((sum, r) => sum + r.resources_left, 0)),
      count: resources.length,
      breakdown: resourceBreakdown(resources),
    }));
}

declare interface SpaceBuildingInfo {
  count: number;
  breakdown: string[];
}

function buildingInfo(buildings: RisqBuilding[]): SpaceBuildingInfo {
  const groups = new Map<string, number>();
  for (const building of buildings) {
    const status = building.under_construction ? ' (under construction)' : '';
    const line = `${building.display_name}${status}`;
    groups.set(line, (groups.get(line) ?? 0) + 1);
  }
  return { count: buildings.length, breakdown: [...groups].map(([line, count]) => `${count} ${line}`) };
}

/** Space content: hex preview, then buildings, gold, owner, unit counts (clickable rows), and resource types */
export class RisqSpaceView {
  private villager_row_button?: RisqSpaceUnitsRowButton;
  private military_row_button?: RisqSpaceUnitsRowButton;
  private shown?: { risq: DwgRisq; space: RisqSpace };
  private unit_rows_visible = false;
  private villager_tooltip = new RisqButtonTooltip(() => villagerTooltipData(this.shown!.risq, this.shown!.space));
  private military_tooltip = new RisqButtonTooltip(() => militaryTooltipData(this.shown!.risq, this.shown!.space));
  private building_hover: RectHoverData = { ps: { x: 0, y: 0 }, pe: { x: 0, y: 0 } };
  private building_info?: SpaceBuildingInfo;
  private building_tooltip = new RisqButtonTooltip(() => this.buildingTooltipData());
  private resource_hovers = new Map<RisqResourceType, RectHoverData>();
  private resource_info = new Map<RisqResourceType, SpaceResourceRow>();
  private resource_tooltips = new Map<RisqResourceType, RisqButtonTooltip>();

  constructor(private open_units_row: (economic: boolean) => void) {}

  private rowButtons(): RisqSpaceUnitsRowButton[] {
    return [this.villager_row_button, this.military_row_button].filter((b): b is RisqSpaceUnitsRowButton => !!b);
  }

  draw(pc: PanelDrawContext, space: RisqSpace, hexagon: RisqSpaceHexagon): void {
    const { ctx, frame, risq } = pc;
    this.shown = { risq, space };
    this.unit_rows_visible = false;
    let yi = frame.yi() + drawName(pc, space.display_name);
    yi += drawSubtitle(pc, `${terrainTypeLabel(space.terrain_type)} space`, yi);
    const separator_distance = 8;
    yi += hexagon.draw(pc, space, separator_distance, yi);
    drawSeparator(pc, yi);
    yi += separator_distance;
    if (pc.visibility < RisqVisibilityLevel.FOG) {
      return;
    }
    const resource_rows = resourceRows(space);
    this.resource_info = new Map(resource_rows.map((row) => [row.type, row]));
    const rows = 5 + resource_rows.length;
    const image_size = Math.min(
      36,
      (1 / rows) * (0.6 * frame.h() - separator_distance - (rows - 1) * separator_distance)
    );
    const row_x = frame.xi() + 0.1 * frame.w();
    const row_w = 0.8 * frame.w();
    ctx.fillStyle = 'black';
    const draw_row_text = (x: number, text: string, max_w: number): void => {
      const row_yc = yi + 0.5 * image_size;
      const font = `bold ${image_size}px serif`;
      ctx.font = font;
      const colon_w = ctx.measureText(': ').width;
      drawText(ctx, ': ', {
        p: { x, y: row_yc },
        w: colon_w,
        fill_style: 'black',
        align: 'left',
        baseline: 'middle',
        font,
      });
      drawText(ctx, text, {
        p: { x: x + colon_w, y: row_yc },
        w: max_w - colon_w,
        fill_style: 'black',
        align: 'left',
        baseline: 'middle',
        font,
      });
    };
    const draw_row = (img: CanvasImageSource, text: string, hover_data?: RectHoverData): void => {
      const ps = { x: row_x, y: yi };
      const pe = { x: ps.x + row_w, y: ps.y + image_size };
      if (hover_data?.hovered) {
        ctx.strokeStyle = 'transparent';
        ctx.fillStyle = hover_data.clicked ? 'rgba(250, 250, 250, 0.4)' : 'rgba(210, 210, 210, 0.25)';
        drawRect(ctx, ps, pe.x - ps.x, pe.y - ps.y);
        ctx.fillStyle = 'black';
      }
      ctx.drawImage(img, ps.x, yi, image_size, image_size);
      draw_row_text(ps.x + image_size + 2, text, row_w - image_size - 2);
      if (!!hover_data) {
        hover_data.ps = ps;
        hover_data.pe = pe;
      }
      yi += image_size + separator_distance;
    };
    this.building_info = space.buildings && buildingInfo([...space.buildings.values()]);
    if (hasBuildings(risq.getGame())) {
      draw_row(risq.getIcon('icons/building64'), space.buildings?.size.toString() ?? '??', this.building_hover);
    } else {
      this.building_info = undefined;
    }
    draw_row(risq.getIcon(resourceTypeImage(RisqResourceType.GOLD)), `${(space.gold_income ?? 0).toFixed(1)}/turn`);
    const owner_color = spaceOwnerColor(space.ownership, risq.getGame()?.players ?? []);
    const owner_name = owner_color
      ? (risq.getGame()?.players[space.ownership ?? -1]?.player.nickname ?? 'Unknown')
      : '--Unclaimed--';
    ctx.fillStyle = owner_color ? owner_color.getString() : 'rgba(255, 255, 255, 0.3)';
    ctx.strokeStyle = 'black';
    ctx.lineWidth = 1;
    drawRect(ctx, { x: row_x, y: yi }, image_size, image_size);
    draw_row_text(row_x + image_size + 2, owner_name, row_w - image_size - 2);
    yi += image_size + separator_distance;
    if (pc.visibility === RisqVisibilityLevel.POOR) {
      const combo_icon = comboUnitIcon(pc);
      if (combo_icon) {
        draw_row(combo_icon, space.unit_count?.toString() ?? '??');
      }
    } else if (pc.visibility >= RisqVisibilityLevel.GOOD) {
      this.unit_rows_visible = true;
      const row_step = image_size + separator_distance;
      if (hasEconomy(risq.getGame())) {
        this.villager_row_button = this.layoutRowButton(
          this.villager_row_button,
          pc.risq.getIcon('icons/villager64'),
          space.num_villager_units?.toString() ?? '??',
          true,
          { x: row_x, y: yi },
          row_w,
          image_size
        );
        yi += row_step;
      }
      this.military_row_button = this.layoutRowButton(
        this.military_row_button,
        pc.risq.getIcon('icons/unit64'),
        space.num_military_units?.toString() ?? '??',
        false,
        { x: row_x, y: yi },
        row_w,
        image_size
      );
      yi += row_step;
    }
    for (const row of resource_rows) {
      draw_row(risq.getIcon(resourceTypeImage(row.type)), `${row.total} (${row.count})`, this.resourceHover(row.type));
    }
  }

  private resourceHover(type: RisqResourceType): RectHoverData {
    const hover = this.resource_hovers.get(type) ?? { ps: { x: 0, y: 0 }, pe: { x: 0, y: 0 } };
    this.resource_hovers.set(type, hover);
    return hover;
  }

  private resourceTooltip(type: RisqResourceType): RisqButtonTooltip {
    const tooltip = this.resource_tooltips.get(type) ?? new RisqButtonTooltip(() => this.resourceTooltipData(type));
    this.resource_tooltips.set(type, tooltip);
    return tooltip;
  }

  private resourceTooltipData(type: RisqResourceType): RisqTooltipData {
    const { total, count, breakdown } = this.resource_info.get(type)!;
    const name = capitalize(RisqResourceType[type]);
    const nodes = `${count} resource${count === 1 ? '' : 's'}`;
    return { title: name, description: `${total} ${name.toLowerCase()} in ${nodes}`, lines: breakdown };
  }

  private buildingTooltipData(): RisqTooltipData {
    const { count, breakdown } = this.building_info!;
    return { title: `${count} Building${count === 1 ? '' : 's'}`, lines: breakdown };
  }

  drawTooltips(ctx: CanvasRenderingContext2D, transform: BoardTransformData, risq: DwgRisq, dt: number): void {
    if (this.building_info) {
      this.building_tooltip.draw(ctx, transform, risq, dt, !!this.building_hover.hovered, false);
    }
    const space = this.shown?.space;
    const unit_rows = [
      [this.villager_row_button, this.villager_tooltip, space?.num_villager_units],
      [this.military_row_button, this.military_tooltip, space?.num_military_units],
    ] as const;
    for (const [button, tooltip, count] of unit_rows) {
      if (this.unit_rows_visible && button && count) {
        tooltip.draw(ctx, transform, risq, dt, button.isHovering(), button.isClicking());
      }
    }
    for (const [type, hover] of this.resource_hovers) {
      if (this.resource_info.has(type)) {
        this.resourceTooltip(type).draw(ctx, transform, risq, dt, !!hover.hovered, false);
      }
    }
  }

  /** Creates the row button on first use, then repositions it and refreshes its count */
  private layoutRowButton(
    button: RisqSpaceUnitsRowButton | undefined,
    icon: HTMLImageElement,
    count: string,
    economic: boolean,
    p: Point2D,
    w: number,
    h: number
  ): RisqSpaceUnitsRowButton {
    const row_button =
      button ?? new RisqSpaceUnitsRowButton({ p, w, h, icon, onOpen: () => this.open_units_row(economic) });
    row_button.setPosition(p);
    row_button.setSize(w, h);
    row_button.setRowText(`: ${count}`);
    return row_button;
  }

  drawRowButtons(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    for (const button of this.rowButtons()) {
      button.draw(ctx, transform, dt);
    }
  }

  mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): void {
    for (const button of this.rowButtons()) {
      button.mousemove(canvas, screen, transform);
    }
    for (const hover of [this.building_hover, ...this.resource_hovers.values()]) {
      const { ps, pe } = hover;
      hover.hovered = screen.x >= ps.x && screen.x <= pe.x && screen.y >= ps.y && screen.y <= pe.y;
    }
  }

  cancelInput(): void {
    this.building_hover.hovered = false;
    this.resource_hovers.forEach((hover) => (hover.hovered = false));
    for (const button of this.rowButtons()) {
      button.setClicking(false);
      button.setHovering(false);
    }
  }

  mousedown(e: MouseEvent): void {
    for (const button of this.rowButtons()) {
      button.mousedown(e);
    }
  }

  mouseup(e: MouseEvent): void {
    for (const button of this.rowButtons()) {
      button.mouseup(e);
    }
  }
}

function drawRegionOwner(pc: PanelDrawContext, region: RisqRegion, yi: number): number {
  const { ctx, frame, risq } = pc;
  const image_size = 40;
  const owner_color = region.owner >= 0 ? risq.getGame()?.players[region.owner]?.color : undefined;
  const owner_name = owner_color
    ? (risq.getGame()?.players[region.owner]?.player.nickname ?? 'Unknown')
    : '--Unclaimed--';
  ctx.fillStyle = owner_color ? owner_color.getString() : 'rgba(255, 255, 255, 0.3)';
  ctx.strokeStyle = 'black';
  ctx.lineWidth = 1;
  drawRect(ctx, { x: frame.xi() + 0.1 * frame.w(), y: yi }, image_size, image_size);
  drawText(ctx, owner_name, {
    p: { x: frame.xi() + 0.1 * frame.w() + image_size + 8, y: yi + 0.5 * image_size },
    w: 0.8 * frame.w() - image_size - 8,
    fill_style: 'black',
    align: 'left',
    baseline: 'middle',
    font: '20px serif',
  });
  return yi + image_size + 12;
}

export function drawRegion(pc: PanelDrawContext, region: RisqRegion): void {
  const { ctx, frame, risq } = pc;
  let yi = frame.yi() + drawName(pc, region.name);
  yi += drawSubtitle(pc, 'Region', yi) + 12;
  drawSeparator(pc, yi);
  yi += 12;
  ctx.drawImage(risq.getIcon(resourceTypeImage(RisqResourceType.GOLD)), frame.xi() + 0.1 * frame.w(), yi, 32, 32);
  drawText(ctx, `${region.gold_bonus} gold / turn`, {
    p: { x: frame.xi() + 0.1 * frame.w() + 40, y: yi + 16 },
    w: 0.8 * frame.w() - 40,
    fill_style: 'black',
    align: 'left',
    baseline: 'middle',
    font: '20px serif',
  });
  yi += 40;
  const space_count = region.has_unexplored_spaces
    ? `${region.spaces.length}/? spaces explored`
    : `${region.spaces.length} space${region.spaces.length === 1 ? '' : 's'}`;
  drawText(ctx, space_count, {
    p: { x: frame.xi() + 0.1 * frame.w(), y: yi },
    w: 0.8 * frame.w(),
    fill_style: 'black',
    align: 'left',
    font: '16px serif',
  });
  yi += 32;
  drawSeparator(pc, yi);
  yi = drawRegionOwner(pc, region, yi + 12);
  drawRegionOwnership(pc, region, yi);
}

export function drawRegionOwnership(pc: PanelDrawContext, region: RisqRegion, yi: number): void {
  const counts = new Map<number, number>();
  for (const key of region.spaces) {
    const owner = pc.risq.session.spaceAt(invertPair(key))?.ownership ?? -1;
    counts.set(owner, (counts.get(owner) ?? 0) + 1);
  }
  const draw_count = (name: string, count: number, color: string = 'black'): void => {
    drawText(pc.ctx, `${name}: ${count} space${count === 1 ? '' : 's'}`, {
      p: { x: pc.frame.xi() + 0.1 * pc.frame.w(), y: yi },
      w: 0.8 * pc.frame.w(),
      fill_style: color,
      align: 'left',
      font: '18px serif',
    });
    yi += 28;
  };
  for (const player of pc.risq.getGame()?.players ?? []) {
    const count = counts.get(player.player.player_id) ?? 0;
    if (count > 0) {
      draw_count(player.player.nickname, count, player.color.getString());
    }
  }
  draw_count('Unowned', counts.get(-1) ?? 0);
}
