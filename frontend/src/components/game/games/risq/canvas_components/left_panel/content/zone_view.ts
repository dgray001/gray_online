import { drawRect, drawText } from '../../../../../util/canvas_util';
import type { RectHoverData, RisqSpace, RisqZone } from '../../../model/types';
import { RisqVisibilityLevel } from '../../../model/types';
import { coordinateToIndex } from '../../../model/coordinates';
import { buildingImage } from '../../../rendering/assets/buildings';
import { resourceIcon } from '../../../rendering/assets/resources';
import { spaceOwnerColor, terrainTypeLabel } from '../../../rendering/terrain';
import type { PanelDrawContext } from './primitives';
import { drawName, drawSeparator, drawSubtitle } from './primitives';
import type { RisqSpaceHexagon } from './space_hexagon';
import { comboUnitIcon } from './space_view';
import { drawUnitImage } from './unit_views';

const MAX_IMAGE_SIZE = 36;
const UNIT_IMAGE_MULTIPLIER = 1.3;

/** Zone content: hex preview with this zone highlighted, its building/resource, owner, and individual units */
export function drawZone(pc: PanelDrawContext, data: { space: RisqSpace; zone: RisqZone }, hexagon: RisqSpaceHexagon) {
  const { ctx, frame, risq } = pc;
  const { space, zone } = data;
  let yi = frame.yi() + drawName(pc, space.display_name);
  const zone_label = zone.terrain_override_display_name ?? terrainTypeLabel(space.terrain_type);
  yi += drawSubtitle(pc, `${zone_label} zone`, yi);
  const separator_distance = 8;
  yi += hexagon.draw(pc, space, separator_distance, yi, coordinateToIndex(1, zone.coordinate));
  drawSeparator(pc, yi);
  yi += separator_distance;
  const units_per_row = Math.floor((0.8 * frame.w() - 1.6 * MAX_IMAGE_SIZE) / (UNIT_IMAGE_MULTIPLIER * MAX_IMAGE_SIZE));
  const economic_rows = Math.ceil(zone.economic_units.length / units_per_row);
  const military_rows = Math.ceil(zone.military_units.length / units_per_row);
  const unit_count_rows = pc.visibility === RisqVisibilityLevel.POOR && !!zone.unit_count ? 1 : 0;
  const rows = 2 + unit_count_rows + economic_rows + military_rows;
  const image_size = Math.min(
    MAX_IMAGE_SIZE,
    (1 / rows) * (0.6 * frame.h() - separator_distance - (rows - 1) * separator_distance)
  );
  ctx.fillStyle = 'black';
  const draw_row = (img: CanvasImageSource, text: string, hover_data?: RectHoverData) => {
    const ps = { x: frame.xi() + 0.1 * frame.w(), y: yi };
    const pe = { x: ps.x + 0.8 * frame.w(), y: ps.y + image_size };
    if (hover_data?.hovered) {
      ctx.strokeStyle = 'transparent';
      ctx.fillStyle = hover_data?.clicked ? 'rgba(250, 250, 250, 0.4)' : 'rgba(210, 210, 210, 0.25)';
      drawRect(ctx, ps, pe.x - ps.x, pe.y - ps.y);
    }
    ctx.drawImage(img, ps.x, ps.y, image_size, image_size);
    drawText(ctx, `: ${text}`, {
      p: { x: ps.x + image_size + 2, y: yi + 0.5 * image_size },
      w: 0.8 * frame.w() - image_size - 2,
      fill_style: 'black',
      align: 'left',
      baseline: 'middle',
      font: `bold ${image_size}px serif`,
    });
    if (!!hover_data) {
      hover_data.ps = ps;
      hover_data.pe = pe;
    }
  };
  if (!!zone.resource) {
    draw_row(resourceIcon(risq, zone.resource), zone.resource.display_name, zone.resource.hover_data);
  } else {
    const building_image = buildingImage(zone.building?.building_id, zone.building?.under_construction);
    const building_color = zone.building ? risq.getGame()?.players[zone.building.player_id]?.color : undefined;
    draw_row(
      building_color ? risq.getPlayerColoredIcon(building_image, building_color) : risq.getIcon(building_image),
      zone.building?.display_name ?? 'Empty Plot',
      zone.building?.hover_data
    );
  }
  yi += image_size + separator_distance;
  const owner_color = spaceOwnerColor(zone.ownership, risq.getGame()?.players ?? []);
  const owner_name =
    owner_color && zone.ownership !== undefined
      ? (risq.getGame()?.players[zone.ownership]?.player.nickname ?? 'Unknown')
      : '--Unclaimed--';
  const owner_ps = { x: frame.xi() + 0.1 * frame.w(), y: yi };
  ctx.fillStyle = owner_color ? owner_color.getString() : 'rgba(255, 255, 255, 0.3)';
  ctx.strokeStyle = 'black';
  ctx.lineWidth = 1;
  drawRect(ctx, owner_ps, image_size, image_size);
  drawText(ctx, `: ${owner_name}`, {
    p: { x: owner_ps.x + image_size + 2, y: yi + 0.5 * image_size },
    w: 0.8 * frame.w() - image_size - 2,
    fill_style: 'black',
    align: 'left',
    baseline: 'middle',
    font: `bold ${image_size}px serif`,
  });
  yi += image_size + separator_distance;
  if (unit_count_rows > 0) {
    const combo_icon = comboUnitIcon(pc);
    if (combo_icon) {
      draw_row(combo_icon, zone.unit_count!.toString());
    }
    yi += image_size + separator_distance;
  }
  const unit_grid = (header_icon: string, unit_ids: number[], grid_rows: number) => {
    if (unit_ids.length === 0) {
      return;
    }
    draw_row(risq.getIcon(header_icon), '');
    const xi = frame.xi() + 0.1 * frame.w() + 0.6 * image_size;
    let i = 1;
    let j = 0;
    for (const u of unit_ids) {
      const unit = zone.units.get(u);
      if (!unit) {
        continue;
      }
      const p = { x: xi + i * UNIT_IMAGE_MULTIPLIER * image_size, y: yi + j * UNIT_IMAGE_MULTIPLIER * image_size };
      drawUnitImage(pc, unit, p, image_size);
      i++;
      if (i > units_per_row) {
        i = 1;
        j++;
      }
    }
    yi += grid_rows * UNIT_IMAGE_MULTIPLIER * image_size + separator_distance;
  };
  unit_grid('icons/villager64', zone.economic_units, economic_rows);
  unit_grid('icons/unit64', zone.military_units, military_rows);
}
