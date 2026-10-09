import type { BoardTransformData } from '../../../../../util/canvas_board/canvas_board';
import { drawRect, drawText } from '../../../../../util/canvas_util';
import type { Point2D } from '../../../../../util/objects2d';
import type { RectHoverData, RisqRegion, RisqSpace } from '../../../model/types';
import { RisqResourceType, RisqVisibilityLevel } from '../../../model/types';
import { resourceTypeImage } from '../../../rendering/assets/resources';
import { COMBO_UNIT_ICON_SIZE, comboUnitIconKey, drawComboUnitIcon } from '../../../rendering/assets/unit';
import { spaceOwnerColor, terrainTypeLabel } from '../../../rendering/terrain';
import { RisqSpaceUnitsRowButton } from '../controls/space_units_row_button';
import type { PanelDrawContext } from './primitives';
import { drawName, drawSeparator, drawSubtitle } from './primitives';
import type { RisqSpaceHexagon } from './space_hexagon';

/** Draws a combo villager+military icon, used where only a total unit count is visible */
export function comboUnitIcon(pc: PanelDrawContext): HTMLCanvasElement | undefined {
  const villager_img = pc.risq.getIcon('icons/villager64');
  const unit_img = pc.risq.getIcon('icons/unit64');
  return pc.risq
    .getImageCache()
    .getImage(comboUnitIconKey(false), COMBO_UNIT_ICON_SIZE, [villager_img, unit_img], (combo_ctx) =>
      drawComboUnitIcon(combo_ctx, villager_img, unit_img)
    );
}

/** Space content: hex preview, then buildings, gold, owner, unit counts (clickable rows), and resource types */
export class RisqSpaceView {
  private villager_row_button?: RisqSpaceUnitsRowButton;
  private military_row_button?: RisqSpaceUnitsRowButton;

  constructor(private open_units_row: (economic: boolean) => void) {}

  private rowButtons(): RisqSpaceUnitsRowButton[] {
    return [this.villager_row_button, this.military_row_button].filter((b): b is RisqSpaceUnitsRowButton => !!b);
  }

  draw(pc: PanelDrawContext, space: RisqSpace, hexagon: RisqSpaceHexagon): void {
    const { ctx, frame, risq } = pc;
    let yi = frame.yi() + drawName(pc, space.display_name);
    yi += drawSubtitle(pc, `${terrainTypeLabel(space.terrain_type)} space`, yi);
    const separator_distance = 8;
    yi += hexagon.draw(pc, space, separator_distance, yi);
    drawSeparator(pc, yi);
    yi += separator_distance;
    if (pc.visibility < RisqVisibilityLevel.POOR) {
      return;
    }
    const rows = 6;
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
    draw_row(risq.getIcon('icons/building64'), space.buildings?.size.toString() ?? '??');
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
    } else {
      const row_step = image_size + separator_distance;
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
    this.drawResourceTypes(pc, space, yi, image_size);
  }

  private drawResourceTypes(pc: PanelDrawContext, space: RisqSpace, yi: number, image_size: number): void {
    const { ctx, frame, risq } = pc;
    const resources = [...(space.total_resources?.entries() ?? [])]
      .filter((r) => r[1] > 0)
      .map((r) => r[0])
      .sort((a, b) => a - b);
    const num_resources = resources.length + 0.3 * (resources.length - 1); // account for slashes
    const resource_image_size = Math.min(image_size, (1.0 / num_resources) * 0.8 * frame.w());
    for (const [i, r] of resources.entries()) {
      if (i > 0) {
        drawText(ctx, '/', {
          p: { x: frame.xi() + 0.1 * frame.w() + (i * 1.3 - 0.15) * resource_image_size, y: yi },
          w: 0.3 * resource_image_size,
          fill_style: 'black',
          align: 'center',
          font: `bold ${resource_image_size}px serif`,
        });
      }
      ctx.drawImage(
        risq.getIcon(resourceTypeImage(r)),
        frame.xi() + 0.1 * frame.w() + i * 1.3 * resource_image_size,
        yi,
        resource_image_size,
        resource_image_size
      );
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
  }

  cancelInput(): void {
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

export function drawRegion(pc: PanelDrawContext, region: RisqRegion): void {
  const { ctx, frame, risq } = pc;
  let yi = frame.yi() + drawName(pc, region.name);
  yi += 12;
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
  yi += image_size + 12;
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
  drawText(ctx, `${region.spaces.length} space${region.spaces.length === 1 ? '' : 's'} explored`, {
    p: { x: frame.xi() + 0.1 * frame.w(), y: yi },
    w: 0.8 * frame.w(),
    fill_style: 'black',
    align: 'left',
    font: '16px serif',
  });
}
