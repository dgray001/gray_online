import type { DwgRisq } from '../../risq';
import type { BattlePlayerStats, ReplayEvent } from '../../model/replay_event';
import { BATTLE_METRICS, battleSides, battleTotals } from '../../model/battle_summary';
import type { BattleMetric } from '../../model/battle_summary';
import type { RisqTurnReport } from '../../transport/turn_report';

type PreviewBattle = (event: ReplayEvent, control: HTMLElement) => void;

export function renderBattleReport(
  risq: DwgRisq,
  report: RisqTurnReport,
  totals: HTMLElement,
  list: HTMLElement,
  empty: HTMLElement,
  preview: PreviewBattle
): void {
  const battles =
    risq.getGame()?.last_turn_journal?.turn_number === report.turn
      ? risq.session.getReplayEvents().filter((event: ReplayEvent): boolean => event.kind === 'battle')
      : [];
  totals.replaceChildren(...(battles.length >= 2 ? [summary(risq, battleTotals(battles), 'Totals')] : []));
  totals.hidden = battles.length < 2;
  list.replaceChildren(
    ...battles.map((event: ReplayEvent, index: number): HTMLLIElement => battleCard(risq, event, index, preview))
  );
  list.hidden = battles.length === 0;
  empty.hidden = battles.length > 0;
  empty.textContent = report.combat.length > 0 ? 'Battle details unavailable.' : 'No battles.';
}

function battleCard(risq: DwgRisq, event: ReplayEvent, index: number, preview: PreviewBattle): HTMLLIElement {
  const item = document.createElement('li');
  const button = document.createElement('button');
  button.type = 'button';
  button.className = 'battle-summary';
  button.appendChild(
    summary(risq, event.battle_stats ?? [], event.title === 'Battle' ? `Battle ${index + 1}` : event.title)
  );
  button.addEventListener('click', (): void => preview(event, button));
  item.appendChild(button);
  return item;
}

function text(class_name: string, value: string): HTMLSpanElement {
  const element = document.createElement('span');
  element.className = class_name;
  element.textContent = value;
  return element;
}

function summary(risq: DwgRisq, stats: BattlePlayerStats[], title: string): HTMLSpanElement {
  const content = text('battle-content', '');
  content.appendChild(text('battle-title', title));
  const sides = battleSides(stats, risq.session.getPlayerId());
  const players = text('battle-players', '');
  for (const side of sides) {
    players.appendChild(playerSide(risq, side));
  }
  content.appendChild(players);
  for (const metric of BATTLE_METRICS) {
    content.appendChild(metricRow(risq, sides, metric));
  }
  return content;
}

function playerColor(risq: DwgRisq, player_id: number): string {
  return risq.getGame()?.players[player_id]?.color.getString() ?? '#777';
}

function playerSide(risq: DwgRisq, stats: BattlePlayerStats[]): HTMLSpanElement {
  const side = text('battle-side', '');
  const names = text('battle-names', '');
  for (const player of stats) {
    const name = text('', risq.getGame()?.players[player.player_id]?.player.nickname ?? `Player ${player.player_id}`);
    name.style.color = playerColor(risq, player.player_id);
    names.appendChild(name);
  }
  side.appendChild(names);
  const counts = text('battle-counts', '');
  counts.appendChild(
    entityCount(
      'unit64',
      'Units',
      stats.reduce((sum: number, player: BattlePlayerStats): number => sum + player.troops, 0)
    )
  );
  counts.appendChild(
    entityCount(
      'building64',
      'Buildings',
      stats.reduce((sum: number, player: BattlePlayerStats): number => sum + player.buildings, 0)
    )
  );
  side.appendChild(counts);
  return side;
}

function entityCount(icon: string, label: string, count: number): HTMLSpanElement {
  const value = text('battle-count', '');
  const image = document.createElement('img');
  image.src = `/images/icons/${icon}.png`;
  image.alt = label;
  image.width = image.height = 24;
  value.append(image, text('', String(count)));
  return value;
}

function metricRow(risq: DwgRisq, sides: BattlePlayerStats[][], metric: BattleMetric): HTMLSpanElement {
  const row = text('battle-metric', '');
  const values = sides.map((side: BattlePlayerStats[]): number =>
    side.reduce((sum: number, player: BattlePlayerStats): number => sum + metric.value(player), 0)
  );
  const label = text('battle-metric-label', '');
  label.append(
    text('', String(Math.round(values[0]))),
    text('', metric.label),
    text('', String(Math.round(values[1])))
  );
  const bar = text('battle-bar', '');
  const total = values[0] + values[1];
  for (const player of sides.flat()) {
    const segment = text('', '');
    segment.style.width = `${total > 0 ? (metric.value(player) * 100) / total : 0}%`;
    segment.style.backgroundColor = playerColor(risq, player.player_id);
    bar.appendChild(segment);
  }
  row.append(label, bar);
  return row;
}
