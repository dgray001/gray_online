import { formatOneDecimal } from '../../../../../../scripts/math';
import { capitalize } from '../../../../../../scripts/util';
import { RisqResourceType } from '../../model/types';
import type { DwgRisq } from '../../risq';
import type { RisqTooltipData } from '../risq_tooltip';

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
