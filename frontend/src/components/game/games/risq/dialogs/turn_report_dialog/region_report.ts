import type { DwgRisq } from '../../risq';
import type { ReplayEvent } from '../../model/replay_event';
import type { RisqReportRegion, RisqTurnReport } from '../../transport/turn_report';

type PreviewRegion = (event: ReplayEvent, control: HTMLElement) => void;

function reportRow(
  risq: DwgRisq,
  row: RisqReportRegion,
  events: readonly ReplayEvent[],
  preview: PreviewRegion
): HTMLLIElement {
  const changed = row.held_start !== row.held_end;
  const change = row.held_end ? 'gained' : 'lost';
  const event = events.find(
    (candidate: ReplayEvent): boolean =>
      candidate.player_id === risq.session.getPlayerId() &&
      candidate.continent?.name === row.name &&
      candidate.continent.change === change
  );
  const status = row.held_end ? ` ${changed ? 'gained' : 'held'} (+${Math.round(row.gold_bonus)} gold/turn)` : ' lost';
  return regionRow(`${row.name}${status}`, row.held_end, changed, event, preview);
}

export function renderRegionReport(
  risq: DwgRisq,
  report: RisqTurnReport,
  list: HTMLElement,
  empty: HTMLElement,
  preview: PreviewRegion
): void {
  const events =
    risq.getGame()?.last_turn_journal?.turn_number === report.turn
      ? risq.session.getReplayEvents().filter((event: ReplayEvent): boolean => event.kind === 'continent')
      : [];
  const rows = report.regions.map((row: RisqReportRegion): HTMLLIElement => reportRow(risq, row, events, preview));
  for (const event of events) {
    if (event.player_id === risq.session.getPlayerId()) {
      continue;
    }
    const player = risq.getGame()?.players[event.player_id]?.player.nickname ?? `Player ${event.player_id}`;
    rows.push(
      regionRow(
        `${player} ${event.continent!.change} ${event.continent!.name}`,
        event.continent!.change === 'gained',
        true,
        event,
        preview
      )
    );
  }
  list.replaceChildren(...rows);
  list.hidden = rows.length === 0;
  empty.hidden = rows.length > 0;
}

function regionRow(
  label: string,
  gained: boolean,
  changed: boolean,
  event: ReplayEvent | undefined,
  preview: PreviewRegion
): HTMLLIElement {
  const item = document.createElement('li');
  item.innerHTML = `<span class="tick ${gained ? 'make' : 'bad'}">${gained ? '+' : '×'}</span>`;
  if (changed) {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'replay-link';
    button.textContent = label;
    button.disabled = !event;
    button.addEventListener('click', (): void => {
      if (event) {
        preview(event, button);
      }
    });
    item.appendChild(button);
  } else {
    item.appendChild(document.createTextNode(label));
  }
  return item;
}
