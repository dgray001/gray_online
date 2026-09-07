import { DialogSize, DwgDialogBox } from '../../../../dialog_box/dialog_box';
import { shortLabel } from '../canvas_components/order_row/order_row_resolve';
import { buildingImage } from '../risq_buildings';
import type { RisqPlayer, RisqProducible, RisqOrderType } from '../risq_data';
import { RisqProducibleKind, RisqResourceType } from '../risq_data';
import { resourceTypeImage } from '../risq_resources';
import { techImage } from '../risq_techs';
import { unitImage } from '../risq_unit';
import type { RisqReportCombatEvent, RisqTurnReport } from '../risq_turn_report';
import { RisqCombatEventKind } from '../risq_turn_report';
import type { DwgRisq } from '../risq';

import html from './turn_report_dialog.html';

import './turn_report_dialog.scss';

const RESOURCE_LABELS: Record<RisqResourceType, string> = {
  [RisqResourceType.ERROR]: 'Error',
  [RisqResourceType.FOOD]: 'Food',
  [RisqResourceType.WOOD]: 'Wood',
  [RisqResourceType.STONE]: 'Stone',
  [RisqResourceType.GOLD]: 'Gold',
};

interface TurnReportDialogData {
  risq: DwgRisq;
  player: RisqPlayer;
  report: RisqTurnReport;
}

function signed(n: number): string {
  if (n > 0) {
    return `+${n}`;
  }
  return n.toString();
}

/** Looks up a producible's display name from the player's current buildings/units, falling back to a generic label */
function producibleName(player: RisqPlayer, kind: RisqProducibleKind, id: number, fallback: string): string {
  const find = (producibles: RisqProducible[]) => producibles.find((p) => p.kind === kind && p.id === id);
  for (const building of player.buildings.values()) {
    const match = find(building.produces);
    if (match) {
      return match.display_name;
    }
  }
  for (const unit of player.units.values()) {
    const match = find(unit.builds);
    if (match) {
      return match.display_name;
    }
  }
  return fallback;
}

function combatEventText(e: RisqReportCombatEvent, viewer_player_id: number): string {
  const mine = e.self_player === viewer_player_id;
  switch (e.kind) {
    case RisqCombatEventKind.UNIT_KILLED:
      return mine ? 'Destroyed an enemy unit' : 'Lost a unit in combat';
    case RisqCombatEventKind.UNIT_LOST:
      return 'Lost a unit in combat';
    case RisqCombatEventKind.BUILDING_RAZED:
      return mine ? 'Razed an enemy building' : 'A building was razed';
    case RisqCombatEventKind.BUILDING_LOST:
      return 'Lost a building';
    default:
      return 'Combat occurred';
  }
}

export class DwgRisqTurnReportDialog extends DwgDialogBox<TurnReportDialogData> {
  private title_heading!: HTMLHeadingElement;
  private scores_body!: HTMLTableSectionElement;
  private land_tiles!: HTMLDivElement;
  private resources_body!: HTMLTableSectionElement;
  private pop_line!: HTMLDivElement;
  private production_lines!: HTMLUListElement;
  private production_empty!: HTMLParagraphElement;
  private combat_lines!: HTMLUListElement;
  private combat_empty!: HTMLParagraphElement;
  private order_summary!: HTMLDivElement;
  private order_failures!: HTMLUListElement;
  private close_button!: HTMLButtonElement;

  private data!: TurnReportDialogData;

  constructor() {
    super();
    this.configureElement('title_heading', 'title');
    this.configureElements(
      'scores_body',
      'land_tiles',
      'resources_body',
      'pop_line',
      'production_lines',
      'production_empty',
      'combat_lines',
      'combat_empty',
      'order_summary',
      'order_failures',
      'close_button'
    );
  }

  override getHTML(): string {
    return html;
  }

  getData(): TurnReportDialogData {
    return this.data;
  }

  setData(data: TurnReportDialogData, parsed?: boolean) {
    this.data = data;
    this.classList.add(`size-${DialogSize.XLARGE}`);
    if (!parsed && !this.fully_parsed) {
      return;
    }
    const { risq, player, report } = data;
    this.close_button.addEventListener('click', () => {
      this.closeDialog();
    });

    this.title_heading.innerText = `Turn ${report.turn} Report`;

    this.renderScores(risq, player, report);
    this.renderLand(report);
    this.renderResources(report);
    this.renderProduction(player, report);
    this.renderCombat(player, report);
    this.renderOrders(report);
  }

  private renderScores(risq: DwgRisq, player: RisqPlayer, report: RisqTurnReport) {
    const rank_of = (key: 'was' | 'now', player_id: number): number => {
      const sorted = [...report.scores].sort((a, b) => b[key] - a[key]);
      return sorted.findIndex((s) => s.player_id === player_id) + 1;
    };
    const rows = [...report.scores]
      .sort((a, b) => b.now - a.now)
      .map((line) => {
        const row = document.createElement('tr');
        const nickname = risq.getGame()?.players[line.player_id]?.player.nickname ?? `Player ${line.player_id}`;
        const color = risq.getGame()?.players[line.player_id]?.color;
        if (line.player_id === player.player.player_id) {
          row.classList.add('you');
        }
        const rank_was = rank_of('was', line.player_id);
        const rank_now = rank_of('now', line.player_id);
        const rank_diff = rank_was - rank_now;
        let rank_html: string;
        if (rank_diff > 0) {
          rank_html = `<span class="rk up">&#9650;&nbsp;+${rank_diff}</span>`;
        } else if (rank_diff < 0) {
          rank_html = `<span class="rk down">&#9660;&nbsp;&minus;${-rank_diff}</span>`;
        } else {
          rank_html = `<span class="rk same">&mdash;</span>`;
        }
        row.innerHTML = `
        <td><span class="swatch" style="background:${color?.getString() ?? 'gray'}"></span>${nickname}</td>
        <td>${line.was}</td>
        <td class="arr">&rarr;</td>
        <td class="now">${line.now}</td>
        <td>${rank_html}</td>
      `;
        return row;
      });
    this.scores_body.replaceChildren(...rows);
  }

  private renderLand(report: RisqTurnReport) {
    const land = report.land;
    const tile = (num: string, cap: string, cls = ''): HTMLDivElement => {
      const el = document.createElement('div');
      el.innerHTML = `<span class="num ${cls}">${num}</span><span class="cap">${cap}</span>`;
      return el;
    };
    const net = land.gained - land.lost;
    this.land_tiles.replaceChildren(
      tile(land.held_start.toString(), 'Held at start'),
      tile(signed(net), net > 0 ? 'Gained' : net < 0 ? 'Lost' : 'Net change', net > 0 ? 'up' : net < 0 ? 'down' : ''),
      tile(land.held_end.toString(), 'Held now'),
      tile(signed(Math.round(land.gold_from_land)), 'Gold from land', 'gold-txt'),
      tile(signed(land.newly_explored), 'Newly explored', land.newly_explored > 0 ? 'up' : '')
    );
  }

  private renderResources(report: RisqTurnReport) {
    const rows = report.resources.map((line) => {
      const row = document.createElement('tr');
      const label = RESOURCE_LABELS[line.resource_type];
      const icon = resourceTypeImage(line.resource_type);
      const spent_html = line.spent > 0 ? `&minus;${Math.round(line.spent)}` : '0';
      row.innerHTML = `
        <td><img src="/images/${icon}.png" alt="" width="16" height="16" />${label}</td>
        <td>${Math.round(line.start)}</td>
        <td class="${line.gathered > 0 ? 'plus' : 'zero'}">${line.gathered > 0 ? '+' : ''}${Math.round(line.gathered)}</td>
        <td class="${line.spent > 0 ? 'minus' : 'zero'}">${spent_html}</td>
        <td class="final">${Math.round(line.final)}</td>
      `;
      return row;
    });
    this.resources_body.replaceChildren(...rows);
  }

  private renderProduction(player: RisqPlayer, report: RisqTurnReport) {
    this.pop_line.innerHTML = `Population <b>${report.population.start} &rarr; ${report.population.end}</b> &nbsp;&middot;&nbsp; cap <b>${report.population.cap_start} &rarr; ${report.population.cap_end}</b>`;
    const lines: HTMLLIElement[] = [];
    const line = (tick_cls: string, tick: string, icon: string, text: string) => {
      const li = document.createElement('li');
      const img = icon ? `<img src="/images/${icon}.png" alt="" width="18" height="18" />` : '';
      li.innerHTML = `<span class="tick ${tick_cls}">${tick}</span>${img}<span>${text}</span>`;
      lines.push(li);
    };
    for (const u of report.production.units_created) {
      const name = producibleName(player, RisqProducibleKind.UNIT, u.unit_id, 'Unit');
      line('make', '+', unitImage(u.unit_id, true), `${u.count > 1 ? `${u.count}× ` : ''}${name} trained`);
    }
    for (const b of report.production.buildings_built) {
      const name = producibleName(player, RisqProducibleKind.BUILDING, b.building_id, 'Building');
      line('make', '+', buildingImage(b.building_id, false, true), `${name} built at (${b.space.x}, ${b.space.y})`);
    }
    for (const tech_id of report.production.techs_researched) {
      const name = producibleName(player, RisqProducibleKind.TECH, tech_id, 'Technology');
      line('tech', '✦', techImage(tech_id), `${name} researched`);
    }
    this.production_lines.replaceChildren(...lines);
    this.production_lines.hidden = lines.length === 0;
    this.production_empty.hidden = lines.length > 0;
  }

  private renderCombat(player: RisqPlayer, report: RisqTurnReport) {
    const lines = report.combat.map((e) => {
      const li = document.createElement('li');
      const mine = e.self_player === player.player.player_id;
      li.innerHTML = `<span class="tick ${mine ? 'make' : 'bad'}">${mine ? '+' : '×'}</span><span>${combatEventText(e, player.player.player_id)} <span class="where">at (${e.space.x}, ${e.space.y})</span></span>`;
      return li;
    });
    this.combat_lines.replaceChildren(...lines);
    this.combat_lines.hidden = lines.length === 0;
    this.combat_empty.hidden = lines.length > 0;
  }

  private renderOrders(report: RisqTurnReport) {
    const orders = report.orders;
    const chip = (n: number, label: string, warn = false): string =>
      `<span${warn && n > 0 ? ' class="warn"' : ''}><b>${n}</b>&nbsp;${label}</span>`;
    this.order_summary.innerHTML = [
      chip(orders.active, 'active'),
      chip(orders.added, 'added'),
      chip(orders.failed, 'failed', true),
      chip(orders.executed, 'executed'),
      chip(orders.cancelled, 'cancelled'),
    ].join('');
    const failures = orders.failures.map((f) => {
      const li = document.createElement('li');
      li.innerHTML = `<span class="tick bad">&times;</span><span>${shortLabel(f.order_type as RisqOrderType)} &mdash; <span class="reason">${f.reason}</span></span>`;
      return li;
    });
    this.order_failures.replaceChildren(...failures);
  }
}

customElements.define('dwg-risq-turn-report-dialog', DwgRisqTurnReportDialog);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-risq-turn-report-dialog': DwgRisqTurnReportDialog;
  }
}
