import { drawText } from '../../../../../util/canvas_util';
import type { RisqBuilding, RisqResource } from '../../../model/types';
import { buildingImage } from '../../../rendering/assets/buildings';
import { resourceIcon, resourceTypeImage } from '../../../rendering/assets/resources';
import type { FoundationDrawData } from '../left_panel_data';
import type { PanelDrawContext } from './primitives';
import { drawHeaderImage, drawName, drawOwnerSeparators, drawSeparator } from './primitives';
import { workersText } from './stats_view';
import { drawGarrisonedUnits } from './unit_views';

export function drawResource(pc: PanelDrawContext, resource: RisqResource) {
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

export function drawBuilding(pc: PanelDrawContext, building: RisqBuilding) {
  const { ctx, frame, risq } = pc;
  let yi = frame.yi() + drawName(pc, building?.display_name ?? 'Empty Plot');
  yi += drawHeaderImage(
    pc,
    yi,
    buildingImage(building?.building_id, building?.under_construction),
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

export function drawFoundation(pc: PanelDrawContext, foundation: FoundationDrawData) {
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
