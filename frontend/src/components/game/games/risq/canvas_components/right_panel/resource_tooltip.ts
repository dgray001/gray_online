import { formatOneDecimal } from '../../../../../../scripts/math';
import { capitalize } from '../../../../../../scripts/util';
import type { GameRisqScoreEntry, RisqPlayer, RisqPlayerResource } from '../../model/types';
import { RisqProducibleKind, RisqResourceType, RisqUnitType } from '../../model/types';
import type { DwgRisq } from '../../risq';
import type { RisqTooltipData } from '../risq_tooltip';
import { drawLine, drawText } from '../../../../util/canvas_util';
import { resourceTypeImage } from '../../rendering/assets/resources';

function signed(amount: number): string {
  const text = formatOneDecimal(amount);
  return amount > 0 && text !== '0' ? `+${text}` : text;
}

function lastTurnLine(risq: DwgRisq, type: RisqResourceType): string | undefined {
  const line = risq.session.getLastTurnReport()?.resources.find((l) => l.resource_type === type);
  if (!line) {
    return undefined;
  }
  const [gathered, income, spent] = [line.gathered, line.income, line.spent].map(Math.round);
  return `Last turn: +${gathered} gathered, +${income} other income, -${spent} spent`;
}

function knownOnMap(risq: DwgRisq, type: RisqResourceType): number {
  const spaces = risq.getGame()?.spaces.flat() ?? [];
  return spaces.reduce((sum, space) => sum + (space?.total_resources?.get(type) ?? 0), 0);
}

function goldIncomeLines(risq: DwgRisq): string[] {
  const player_id = risq.getPlayerId();
  const game = risq.getGame();
  const owned = (game?.spaces.flat() ?? []).filter((space) => space?.ownership === player_id);
  const land = owned.reduce((sum, space) => sum + (space?.gold_income ?? 0), 0);
  const regions = (game?.regions ?? []).filter((r) => r.owner === player_id);
  const bonuses = regions.reduce((sum, region) => sum + region.gold_bonus, 0);
  return [`Land income: ${signed(land)}/turn`, `Region bonuses: ${signed(bonuses)}/turn`];
}

export function resourceTooltipData(risq: DwgRisq, type: RisqResourceType): RisqTooltipData {
  const title = capitalize(RisqResourceType[type]);
  const resource = risq.getPlayer()?.resources.get(type);
  if (!resource) {
    return { title };
  }
  const { amount, spending, workers, predicted_workers } = resource;
  const worker_text = workers === predicted_workers ? `${workers}` : `${workers} (${predicted_workers} after orders)`;
  const lines = [
    `Stockpile: ${formatOneDecimal(amount)}`,
    `Pending orders: ${signed(-spending)}`,
    risq.getGame()?.has_economy === false ? undefined : `Workers gathering: ${worker_text}`,
    lastTurnLine(risq, type),
    `Known on map: ${Math.round(knownOnMap(risq, type))}`,
    ...(type === RisqResourceType.GOLD ? goldIncomeLines(risq) : []),
  ];
  return { title, lines: lines.filter((line): line is string => line !== undefined) };
}

function unitTypes(player: RisqPlayer): Map<number, RisqUnitType> {
  const types = new Map<number, RisqUnitType>();
  for (const unit of player.units.values()) {
    types.set(unit.unit_id, unit.unit_type);
  }
  const producibles = [...player.buildings.values()].flatMap((b) => b.produces).concat(player.available_mercenaries);
  for (const producible of producibles) {
    if (producible.kind === RisqProducibleKind.UNIT && producible.unit_type !== undefined) {
      types.set(producible.id, producible.unit_type);
    }
  }
  return types;
}

export function populationTooltipData(risq: DwgRisq): RisqTooltipData {
  const player = risq.getPlayer();
  if (!player) {
    return { title: 'Population' };
  }
  const current = new Map<RisqUnitType, number>();
  for (const unit of player.units.values()) {
    current.set(unit.unit_type, (current.get(unit.unit_type) ?? 0) + 1);
  }
  const queued = new Map<RisqUnitType, number>();
  const types = unitTypes(player);
  for (const [id, count] of risq.planning.queuedUnitCounts()) {
    const type = types.get(id) ?? RisqUnitType.NONE;
    queued.set(type, (queued.get(type) ?? 0) + count);
  }
  const queued_total = [...queued.values()].reduce((sum, count) => sum + count, 0);
  const ids = [...new Set([...current.keys(), ...queued.keys()])].sort((a, b) => a - b);
  const type_lines = ids.map((id) => {
    const extra = queued.get(id);
    return `${capitalize(RisqUnitType[id].toLowerCase())}: ${current.get(id) ?? 0}${extra ? ` +${extra}` : ''}`;
  });
  const limit = player.unlimited_population ? '' : ` / ${player.population_limit}`;
  const total = `Total: ${player.units.size}${queued_total ? ` +${queued_total}` : ''}${limit}`;
  return { title: 'Population', lines: [total, ...type_lines] };
}

export interface RightPanelGeometry {
  xi: number;
  xf: number;
  w: number;
}

export function drawSeparator(ctx: CanvasRenderingContext2D, geom: RightPanelGeometry, yi: number): void {
  ctx.strokeStyle = 'rgba(60, 60, 60, 0.6)';
  ctx.lineWidth = 2;
  drawLine(ctx, { x: geom.xi + 0.15 * geom.w, y: yi }, { x: geom.xf - 0.15 * geom.w, y: yi });
}

export function resourceRowAt(
  resource_row_ys: Iterable<[RisqResourceType, number]>,
  screen_y: number
): RisqResourceType | undefined {
  return [...resource_row_ys].find(([, yi]) => screen_y >= yi && screen_y < yi + 30)?.[0];
}

export function drawPopulationRow(
  ctx: CanvasRenderingContext2D,
  risq: DwgRisq,
  geom: RightPanelGeometry,
  yi: number,
  pop: number,
  pending: number,
  limit: number | undefined
): void {
  ctx.beginPath();
  ctx.drawImage(risq.getIcon('icons/person64'), geom.xi + 0.1 * geom.w, yi, 30, 30);
  const count = pending > 0 ? `${pop}+${pending}` : `${pop}`;
  drawText(ctx, limit === undefined ? count : `${count}/${limit}`, {
    p: { x: geom.xi + 0.1 * geom.w + 36, y: yi + 15 },
    w: 0.9 * geom.w - 36,
    fill_style: limit !== undefined && pop + pending >= limit ? 'rgb(200, 30, 30)' : 'black',
    baseline: 'middle',
  });
}

export function drawResourceRow(
  ctx: CanvasRenderingContext2D,
  risq: DwgRisq,
  geom: RightPanelGeometry,
  yi: number,
  rt: RisqResourceType,
  pr: RisqPlayerResource,
  workers_w: number,
  padding: number
): void {
  ctx.beginPath();
  ctx.drawImage(risq.getIcon(resourceTypeImage(rt)), geom.xi + 0.1 * geom.w, yi, 30, 30);
  const net = pr.amount - pr.spending;
  const sign = pr.spending > 0 ? '-' : '+';
  const spending = formatOneDecimal(Math.abs(pr.spending));
  const text = pr.spending !== 0 ? `${formatOneDecimal(net)} (${sign}${spending})` : formatOneDecimal(net);
  drawText(ctx, text, {
    p: { x: geom.xi + 0.1 * geom.w + 36, y: yi + 15 },
    w: 0.9 * geom.w - 36 - workers_w,
    fill_style: net < 0 ? 'rgb(200, 30, 30)' : 'black',
    baseline: 'middle',
    font: '18px serif',
  });
  if (risq.getGame()?.has_economy === false) {
    return;
  }
  const delta = pr.predicted_workers - pr.workers;
  const workers = `${pr.workers}${delta > 0 ? `+${delta}` : delta < 0 ? delta : ''}`;
  const workers_x = geom.xf - padding - workers_w;
  ctx.drawImage(risq.getIcon('icons/villager64'), workers_x, yi + 5, 20, 20);
  drawText(ctx, workers, {
    p: { x: workers_x + 24, y: yi + 15 },
    w: workers_w - 24,
    fill_style: 'black',
    align: 'left',
    baseline: 'middle',
    font: '16px serif',
  });
}

export function drawScoreRow(
  ctx: CanvasRenderingContext2D,
  geom: RightPanelGeometry,
  yi: number,
  score: GameRisqScoreEntry
): void {
  ctx.beginPath();
  const text = `${score.nickname}: ${score.score}`;
  drawText(ctx, text, {
    p: { x: geom.xi + 0.9 * geom.w, y: yi + 15 },
    w: 0.8 * geom.w,
    fill_style: score.color.getString(),
    baseline: 'middle',
    align: 'right',
    font: '18px serif',
  });
  if (score.eliminated) {
    const text_w = Math.min(ctx.measureText(text).width, 0.8 * geom.w);
    ctx.strokeStyle = score.color.getString();
    ctx.lineWidth = 2;
    drawLine(ctx, { x: geom.xi + 0.9 * geom.w - text_w, y: yi + 15 }, { x: geom.xi + 0.9 * geom.w, y: yi + 15 });
  }
}
