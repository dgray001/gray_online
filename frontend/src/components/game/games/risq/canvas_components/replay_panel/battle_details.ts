import type { BattlePlayerStats, ReplayEvent } from '../../model/replay_event';
import type { DwgRisq } from '../../risq';

type BattleSide = { stats: BattlePlayerStats[]; x: number; width: number };
type BattleMetric = { label: string; value: (stats: BattlePlayerStats) => number };
const METRICS: BattleMetric[] = [
  { label: 'Kills', value: (stats: BattlePlayerStats): number => stats.kills },
  { label: 'Razes', value: (stats: BattlePlayerStats): number => stats.razes },
  { label: 'Damage', value: (stats: BattlePlayerStats): number => stats.unit_damage + stats.building_damage },
];

export function drawBattleDetails(
  ctx: CanvasRenderingContext2D,
  risq: DwgRisq,
  event: ReplayEvent,
  x: number,
  y: number,
  width: number
): void {
  const stats = event.battle_stats ?? [];
  const player_id = risq.session.getPlayerId();
  const sides: BattleSide[] = [
    {
      stats: stats.filter((entry: BattlePlayerStats): boolean => entry.player_id === player_id),
      x,
      width: (width - 24) / 2,
    },
    {
      stats: stats.filter((entry: BattlePlayerStats): boolean => entry.player_id !== player_id),
      x: x + width / 2 + 12,
      width: (width - 24) / 2,
    },
  ];
  ctx.textBaseline = 'alphabetic';
  const name_rows = Math.max(...sides.map((side: BattleSide): number => side.stats.length), 1);
  for (const side of sides) {
    drawSide(ctx, risq, side, y, name_rows);
  }
  const header_height = 56 + name_rows * 20;
  METRICS.forEach((metric: BattleMetric, index: number): void =>
    drawMetric(ctx, risq, sides, metric, y + header_height + index * 48)
  );
}

function drawMetric(
  ctx: CanvasRenderingContext2D,
  risq: DwgRisq,
  sides: BattleSide[],
  metric: BattleMetric,
  y: number
): void {
  const values = sides.map((side: BattleSide): number =>
    side.stats.reduce((sum: number, stats: BattlePlayerStats): number => sum + metric.value(stats), 0)
  );
  const x = sides[0].x;
  const width = sides[1].x + sides[1].width - x;
  const total = values[0] + values[1];
  ctx.fillStyle = 'black';
  ctx.fillText(metric.label, x + width / 2, y);
  ctx.fillText(String(Math.round(values[0])), x + 12, y);
  ctx.fillText(String(Math.round(values[1])), x + width - 12, y);
  ctx.fillStyle = '#a59076';
  ctx.fillRect(x, y + 8, width, 16);
  let offset = x;
  for (const stats of sides.flatMap((side: BattleSide): BattlePlayerStats[] => side.stats)) {
    const segment = total > 0 ? (width * metric.value(stats)) / total : 0;
    ctx.fillStyle = playerColor(risq, stats.player_id);
    ctx.fillRect(offset, y + 8, segment, 16);
    offset += segment;
  }
}

function playerColor(risq: DwgRisq, player_id: number): string {
  return risq.getGame()?.players[player_id]?.color.getString() ?? '#777';
}

function drawSide(ctx: CanvasRenderingContext2D, risq: DwgRisq, side: BattleSide, y: number, name_rows: number): void {
  ctx.font = 'bold 16px sans-serif';
  ctx.textAlign = 'center';
  side.stats.forEach((stats: BattlePlayerStats, index: number): void => {
    ctx.fillStyle = playerColor(risq, stats.player_id);
    ctx.fillText(
      risq.getGame()?.players[stats.player_id]?.player.nickname ?? '',
      side.x + side.width / 2,
      y + 16 + index * 20,
      side.width
    );
  });
  const count_y = y + name_rows * 20 + 4;
  const troops = side.stats.reduce((sum: number, stats: BattlePlayerStats): number => sum + stats.troops, 0);
  const buildings = side.stats.reduce((sum: number, stats: BattlePlayerStats): number => sum + stats.buildings, 0);
  ctx.fillStyle = 'black';
  ctx.drawImage(risq.getIcon('icons/unit64'), side.x + side.width / 2 - 52, count_y, 24, 24);
  ctx.fillText(String(troops), side.x + side.width / 2 - 18, count_y + 18);
  ctx.drawImage(risq.getIcon('icons/building64'), side.x + side.width / 2, count_y, 24, 24);
  ctx.fillText(String(buildings), side.x + side.width / 2 + 34, count_y + 18);
}
