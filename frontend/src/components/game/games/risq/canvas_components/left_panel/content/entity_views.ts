import { drawRect, drawText } from '../../../../../util/canvas_util';
import { drawUnitCountBadge } from '../../../rendering/unit_count_badge';
import type { RisqBuilding, RisqResource } from '../../../model/types';
import { buildingImage } from '../../../rendering/assets/buildings';
import { resourceIcon, resourceTypeImage } from '../../../rendering/assets/resources';
import type { FoundationDrawData } from '../left_panel_data';
import { ACTION_GRID_COLS, PANEL_PADDING } from '../layout';
import type { PanelDrawContext } from './primitives';
import { drawHeaderImage, drawName, drawOwnerSeparators, drawSeparator } from './primitives';
import { workersText } from './stats_view';
import { drawGarrisonedUnits } from './unit_views';

export function drawResource(pc: PanelDrawContext, resource: RisqResource): void {
  const { ctx, frame, risq } = pc;
  let yi = frame.yi() + drawName(pc, resource.display_name);
  yi += drawHeaderImage(pc, yi, resourceIcon(risq, resource));
  drawSeparator(pc, yi);
  yi += 8;
  ctx.beginPath();
  ctx.drawImage(risq.getIcon(resourceTypeImage(resource)), frame.xi() + 0.1 * frame.w(), yi, 40, 40);
  drawText(ctx, resource.resources_left.toFixed(1), {
    p: { x: frame.xi() + 0.1 * frame.w() + 48, y: yi + 20 },
    w: 0.9 * frame.w() - 48,
    fill_style: 'black',
    baseline: 'middle',
    font: '36px serif',
  });
  yi += 50;
  drawText(ctx, `Base gather speed: ${resource.base_gather_speed}`, {
    p: { x: frame.xi() + 0.1 * frame.w(), y: yi + 12 },
    w: 0.9 * frame.w(),
    fill_style: 'black',
    baseline: 'middle',
    font: '18px serif',
  });
  const full = risq.planning.gatherers(resource).predicted.size >= resource.gather_capacity;
  drawText(ctx, `Workers: ${workersText(risq, resource, resource.gather_capacity)}`, {
    p: { x: frame.xi() + 0.1 * frame.w(), y: yi + 36 },
    w: 0.9 * frame.w(),
    fill_style: full ? 'rgb(200, 30, 30)' : 'black',
    baseline: 'middle',
    font: '18px serif',
  });
}

export function drawBuildings(pc: PanelDrawContext, buildings: RisqBuilding[]): void {
  const groups = new Map<number, RisqBuilding[]>();
  for (const building of buildings) {
    groups.set(building.building_id, [...(groups.get(building.building_id) ?? []), building]);
  }
  const content_y = pc.frame.yi() + drawName(pc, `${buildings.length} Buildings`);
  const s = pc.layout.grid_s;
  const capacity =
    Math.floor((pc.layout.separator_below_stats - PANEL_PADDING - content_y) / (s + PANEL_PADDING)) * ACTION_GRID_COLS;
  const blocks =
    buildings.length <= capacity
      ? buildings.map((building: RisqBuilding): RisqBuilding[] => [building])
      : groups.size <= capacity
        ? [...groups.values()]
        : [buildings];
  for (const [i, group] of blocks.entries()) {
    const building = group[0];
    const p = {
      x: pc.layout.grid_x0 + (i % ACTION_GRID_COLS) * (s + PANEL_PADDING),
      y: pc.layout.separator_below_stats - (Math.floor(i / ACTION_GRID_COLS) + 1) * (s + PANEL_PADDING),
    };
    const image = buildingImage(building.building_id, building.under_construction, false, building.combat_stats);
    pc.ctx.drawImage(
      pc.risq.getPlayerColoredIcon(image, pc.risq.getGame()!.players[building.player_id].color),
      p.x,
      p.y,
      s,
      s
    );
    if (group.length > 1) {
      drawUnitCountBadge(pc.ctx, group.length, p, s);
    }
    pc.ctx.fillStyle = pc.buildingTile?.(group, p, s) ?? 'transparent';
    pc.ctx.strokeStyle = 'transparent';
    drawRect(pc.ctx, p, s, s);
  }
  drawOwnerSeparators(pc, buildings[0].player_id);
}

export function drawBuilding(pc: PanelDrawContext, building: RisqBuilding): void {
  const { ctx, frame, risq } = pc;
  let yi = frame.yi() + drawName(pc, building?.display_name ?? 'Empty Plot');
  yi += drawHeaderImage(
    pc,
    yi,
    buildingImage(building?.building_id, building?.under_construction, false, building?.combat_stats),
    building ? risq.getGame()?.players[building.player_id]?.color : undefined
  );
  drawSeparator(pc, yi);
  if (!building) {
    yi += 12;
    drawText(ctx, 'An empty lot that can be built on', {
      p: { x: frame.xi() + 0.1 * frame.w(), y: yi },
      w: 0.9 * frame.w(),
      fill_style: 'black',
      align: 'left',
      font: `14px serif`,
    });
    return;
  }
  pc.stats.draw(pc, frame.yi() + 0.25 * frame.h() + 6, building, building);
  if (drawOwnerSeparators(pc, building.player_id) && pc.layout.garrison_rows > 0) {
    drawGarrisonedUnits(pc, building);
  }
}

export function drawFoundation(pc: PanelDrawContext, foundation: FoundationDrawData): void {
  const { ctx, frame, risq } = pc;
  let yi = frame.yi() + drawName(pc, foundation.display_name);
  yi += drawHeaderImage(pc, yi, 'risq/buildings/construction', risq.getGame()?.players[foundation.player_id]?.color);
  drawSeparator(pc, yi);
  yi += 12;
  drawText(ctx, 'A planned foundation awaiting construction units', {
    p: { x: frame.xi() + 0.1 * frame.w(), y: yi },
    w: 0.9 * frame.w(),
    fill_style: 'black',
    align: 'left',
    font: `14px serif`,
  });
  const stamina =
    risq.planning.getLocalFoundation(foundation.coordinate_key)?.stamina_cost ??
    risq.getPlayer()?.planned_foundations.get(foundation.coordinate_key)?.stamina_cost;
  if (stamina !== undefined) {
    drawText(ctx, `Construction progress: 0 / ${stamina}`, {
      p: { x: frame.xi() + 0.1 * frame.w(), y: yi + 36 },
      w: 0.9 * frame.w(),
      fill_style: 'black',
      baseline: 'middle',
      font: '18px serif',
    });
  }
}
