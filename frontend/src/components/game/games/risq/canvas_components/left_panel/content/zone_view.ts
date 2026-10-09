import { drawRect, drawText } from '../../../../../util/canvas_util';
import type { RectHoverData, RisqSpace, RisqZone } from '../../../model/types';
import { RisqVisibilityLevel } from '../../../model/types';
import { coordinateToIndex } from '../../../model/coordinates';
import { buildingImage, rubbleImage } from '../../../rendering/assets/buildings';
import { resourceIcon } from '../../../rendering/assets/resources';
import { spaceOwnerColor, terrainTypeLabel } from '../../../rendering/terrain';
import type { PanelDrawContext } from './primitives';
import { drawName, drawSeparator, drawSubtitle } from './primitives';
import type { RisqSpaceHexagon } from './space_hexagon';
import { comboUnitIcon } from './space_view';
import { drawUnitImage } from './unit_views';
import { PANEL_PADDING } from '../layout';

const MAX_IMAGE_SIZE = 36;
const UNIT_IMAGE_MULTIPLIER = 1.3;
const UNIT_COUNT_FONT_SIZE = 14;

/** Zone content: hex preview with this zone highlighted, its building/resource, owner, and individual units */
export function drawZone(
  pc: PanelDrawContext,
  data: { space: RisqSpace; zone: RisqZone },
  hexagon: RisqSpaceHexagon
): void {
  const { ctx, frame, risq } = pc;
  const { space, zone } = data;
  let yi = frame.yi() + drawName(pc, space.display_name);
  const zone_label = zone.terrain_override_display_name ?? terrainTypeLabel(space.terrain_type);
  yi += drawSubtitle(pc, `${zone_label} zone`, yi);
  const separator_distance = 8;
  yi += hexagon.draw(pc, space, separator_distance, yi, coordinateToIndex(1, zone.coordinate));
  drawSeparator(pc, yi);
  yi += separator_distance;

  const ICON_SIZE = MAX_IMAGE_SIZE;
  ctx.font = `bold ${ICON_SIZE}px serif`;
  ctx.textAlign = 'left';
  ctx.direction = 'ltr';
  const colon_width = ctx.measureText(':').actualBoundingBoxRight;
  const content_start_x = frame.xi() + 0.1 * frame.w() + ICON_SIZE + PANEL_PADDING + colon_width + PANEL_PADDING;
  const draw_colon = (): void => {
    ctx.font = `bold ${ICON_SIZE}px serif`;
    ctx.fillStyle = 'black';
    ctx.textAlign = 'left';
    ctx.textBaseline = 'middle';
    ctx.fillText(':', frame.xi() + 0.1 * frame.w() + ICON_SIZE + PANEL_PADDING, yi + 0.5 * ICON_SIZE);
  };

  const unit_count_rows = pc.visibility === RisqVisibilityLevel.POOR && !!zone.unit_count ? 1 : 0;
  const fixed_rows = 2 + unit_count_rows;
  const available_w = frame.xi() + 0.9 * frame.w() - content_start_x;
  const available_h = 0.6 * frame.h() - separator_distance - fixed_rows * (ICON_SIZE + separator_distance);
  const total_units = zone.economic_units.length + zone.military_units.length;

  let unit_size = MAX_IMAGE_SIZE;
  let units_per_row = Math.max(1, Math.floor(available_w / (UNIT_IMAGE_MULTIPLIER * MAX_IMAGE_SIZE)));

  if (total_units > 0) {
    unit_size = 0;
    for (let cols = 1; cols <= total_units; cols++) {
      const rows = Math.ceil(zone.economic_units.length / cols) + Math.ceil(zone.military_units.length / cols);
      const size = Math.min(
        MAX_IMAGE_SIZE,
        available_w / (cols * UNIT_IMAGE_MULTIPLIER),
        available_h / (rows * UNIT_IMAGE_MULTIPLIER)
      );
      if (size >= unit_size) {
        unit_size = size;
        units_per_row = cols;
      }
    }
  }

  const economic_rows = Math.ceil(zone.economic_units.length / units_per_row);
  const military_rows = Math.ceil(zone.military_units.length / units_per_row);

  ctx.fillStyle = 'black';
  const draw_row = (img: CanvasImageSource, text: string, hover_data?: RectHoverData, icon_only = false): void => {
    const ps = { x: frame.xi() + 0.1 * frame.w(), y: yi };
    const pe = {
      x: ps.x + (icon_only ? ICON_SIZE : 0.8 * frame.w()),
      y: ps.y + (icon_only ? ICON_SIZE : Math.max(ICON_SIZE, text ? 0 : unit_size)),
    };
    if (hover_data?.hovered) {
      ctx.strokeStyle = 'transparent';
      ctx.fillStyle = hover_data?.clicked ? 'rgba(250, 250, 250, 0.4)' : 'rgba(210, 210, 210, 0.25)';
      drawRect(ctx, ps, pe.x - ps.x, pe.y - ps.y);
    }
    ctx.drawImage(img, ps.x, ps.y, ICON_SIZE, ICON_SIZE);

    draw_colon();

    if (text) {
      drawText(ctx, text, {
        p: { x: content_start_x, y: yi + 0.5 * ICON_SIZE },
        w: frame.xi() + 0.9 * frame.w() - content_start_x,
        fill_style: 'black',
        align: 'left',
        baseline: 'middle',
        font: `bold ${ICON_SIZE}px serif`,
      });
    }
    if (!!hover_data) {
      hover_data.ps = ps;
      hover_data.pe = pe;
    }
  };

  if (!!zone.resource) {
    draw_row(resourceIcon(risq, zone.resource), zone.resource.display_name, zone.resource.hover_data);
  } else {
    const building_image =
      !zone.building && zone.destroyed_building
        ? rubbleImage(zone.destroyed_building, zone.destroyed_building_turns ?? 1)
        : buildingImage(
            zone.building?.building_id,
            zone.building?.under_construction,
            false,
            zone.building?.combat_stats
          );
    const building_color = zone.building ? risq.getGame()?.players[zone.building.player_id]?.color : undefined;
    draw_row(
      building_color ? risq.getPlayerColoredIcon(building_image, building_color) : risq.getIcon(building_image),
      zone.building?.display_name ?? (zone.destroyed_building ? 'Rubble' : 'Empty Plot'),
      zone.building?.hover_data
    );
  }
  yi += ICON_SIZE + separator_distance;

  const owner_color = spaceOwnerColor(zone.ownership, risq.getGame()?.players ?? []);
  const owner_name =
    owner_color && zone.ownership !== undefined
      ? (risq.getGame()?.players[zone.ownership]?.player.nickname ?? 'Unknown')
      : '--Unclaimed--';
  const owner_ps = { x: frame.xi() + 0.1 * frame.w(), y: yi };
  ctx.fillStyle = owner_color ? owner_color.getString() : 'rgba(255, 255, 255, 0.3)';
  ctx.strokeStyle = 'black';
  ctx.lineWidth = 1;
  drawRect(ctx, owner_ps, ICON_SIZE, ICON_SIZE);
  draw_colon();
  drawText(ctx, owner_name, {
    p: { x: content_start_x, y: yi + 0.5 * ICON_SIZE },
    w: frame.xi() + 0.9 * frame.w() - content_start_x,
    fill_style: 'black',
    align: 'left',
    baseline: 'middle',
    font: `bold ${ICON_SIZE}px serif`,
  });
  yi += ICON_SIZE + separator_distance;

  if (unit_count_rows > 0) {
    const combo_icon = comboUnitIcon(pc);
    if (combo_icon) {
      draw_row(combo_icon, zone.unit_count!.toString());
    }
    yi += ICON_SIZE + separator_distance;
  }

  const unit_grid = (
    header_icon: string,
    unit_ids: number[],
    grid_rows: number,
    hover_key: 'economic_hover_data' | 'military_hover_data'
  ): void => {
    if (unit_ids.length === 0) {
      return;
    }
    if (!zone[hover_key]) {
      zone[hover_key] = { hovered: false, clicked: false, ps: { x: 0, y: 0 }, pe: { x: 0, y: 0 } };
    }
    draw_row(risq.getIcon(header_icon), '', zone[hover_key], true);
    if (grid_rows > 1) {
      drawText(ctx, `(${unit_ids.length})`, {
        p: { x: frame.xi() + 0.1 * frame.w() + 0.5 * ICON_SIZE, y: yi + ICON_SIZE + PANEL_PADDING },
        w: ICON_SIZE,
        fill_style: 'black',
        align: 'center',
        baseline: 'top',
        font: `${UNIT_COUNT_FONT_SIZE}px serif`,
      });
    }

    let i = 0;
    let j = 0;
    for (const u of unit_ids) {
      const unit = zone.units.get(u);
      if (!unit) {
        continue;
      }
      const p = {
        x: content_start_x + i * UNIT_IMAGE_MULTIPLIER * unit_size,
        y: yi + j * UNIT_IMAGE_MULTIPLIER * unit_size,
      };
      drawUnitImage(pc, unit, p, unit_size);
      i++;
      if (i >= units_per_row) {
        i = 0;
        j++;
      }
    }
    const header_height = ICON_SIZE + (grid_rows > 1 ? PANEL_PADDING + UNIT_COUNT_FONT_SIZE : 0);
    yi += Math.max(header_height, grid_rows * UNIT_IMAGE_MULTIPLIER * unit_size) + separator_distance;
  };

  unit_grid('icons/villager64', zone.economic_units, economic_rows, 'economic_hover_data');
  unit_grid('icons/unit64', zone.military_units, military_rows, 'military_hover_data');
}
