import { apiGet } from '../../../../../scripts/api';
import { DialogSize, DwgDialogBox } from '../../../../dialog_box/dialog_box';
import { buildingImage } from '../risq_buildings';
import type { RisqCost } from '../risq_data';
import { RisqProducibleKind, RisqRange, RisqResourceType, RisqUnitType } from '../risq_data';
import { resourceTypeImage } from '../risq_resources';
import { techImage } from '../risq_techs';
import { unitImage } from '../risq_unit';
import type { DwgRisq } from '../risq';

import html from './tech_tree_dialog.html';

import './tech_tree_dialog.scss';

interface RisqUnitStatsEntry {
  health: number;
  attack_blunt: number;
  attack_piercing: number;
  attack_range: RisqRange;
  defense_blunt: number;
  defense_piercing: number;
  penetration_blunt: number;
  penetration_piercing: number;
}

interface RisqTechBonusEntry {
  max_health: number;
  turn_stamina: number;
  attack_blunt: number;
  attack_piercing: number;
  attack_magic: number;
  defense_blunt: number;
  defense_piercing: number;
  defense_magic: number;
  penetration_blunt: number;
  penetration_piercing: number;
  penetration_magic: number;
}

interface RisqProducibleEntry {
  row: number;
  col: number;
  kind: RisqProducibleKind;
  id: number;
  cost: RisqCost;
  stamina_cost: number;
  display_name: string;
  description: string;
  required_tech_id: number;
  stats?: RisqUnitStatsEntry;
  bonus?: RisqTechBonusEntry;
  affects_unit_ids?: number[];
  affects_unit_types?: RisqUnitType[];
}

interface RisqBuildingTreeEntry {
  building_id: number;
  display_name: string;
  description: string;
  required_tech_id: number;
  produces: RisqProducibleEntry[];
  cost: RisqCost;
  stamina_cost: number;
  stats?: RisqUnitStatsEntry;
}

interface TechTreeDialogData {
  risq: DwgRisq;
}

type TreeStatus = 'researched' | 'available' | 'locked';

interface HeaderNode {
  kind: 'header';
  x: number;
  y: number;
  w: number;
  h: number;
  status: TreeStatus;
  building: RisqBuildingTreeEntry;
}

interface ProducibleNode {
  kind: 'producible';
  x: number;
  y: number;
  w: number;
  h: number;
  status: TreeStatus;
  entry: RisqProducibleEntry;
}

type TreeNode = HeaderNode | ProducibleNode;

const PADDING = 24;
const COLUMN_W = 220;
const COLUMN_GAP = 44;
const HEADER_H = 74;
const HEADER_TO_NODES_GAP = 24;
const NODE_S = 56;
const NODE_GAP = 10;
const ICON_S_HEADER = 40;
const ICON_PADDING_NODE = 9;

function techStatus(
  required_tech_id: number,
  researched: boolean,
  researched_techs: Map<number, boolean> | undefined
): TreeStatus {
  if (researched) {
    return 'researched';
  }
  if (required_tech_id !== 0 && !researched_techs?.get(required_tech_id)) {
    return 'locked';
  }
  return 'available';
}

function nodeIcon(entry: RisqProducibleEntry): string {
  switch (entry.kind) {
    case RisqProducibleKind.UNIT:
      return unitImage(entry.id, true);
    case RisqProducibleKind.BUILDING:
      return buildingImage(entry.id, false, true);
    case RisqProducibleKind.TECH:
      return techImage(entry.id);
    default:
      return '';
  }
}

const STATUS_FILL: Record<TreeStatus, string> = {
  researched: 'rgba(80, 200, 80, 0.2)',
  available: 'rgba(255, 255, 255, 0.08)',
  locked: 'rgba(255, 255, 255, 0.03)',
};
const STATUS_STROKE: Record<TreeStatus, string> = {
  researched: 'rgb(80, 200, 80)',
  available: 'rgb(200, 190, 175)',
  locked: 'rgb(90, 90, 90)',
};
const STATUS_TEXT: Record<TreeStatus, string> = {
  researched: 'rgb(230, 230, 230)',
  available: 'rgb(230, 230, 230)',
  locked: 'rgb(110, 110, 110)',
};

const RANGE_DISTANCE: Partial<Record<RisqRange, number>> = {
  [RisqRange.SPACE]: 0,
  [RisqRange.ADJACENT]: 1,
  [RisqRange.SECONDARY]: 2,
};

export class DwgRisqTechTreeDialog extends DwgDialogBox<TechTreeDialogData> {
  private title_heading!: HTMLHeadingElement;
  private canvas_scroll!: HTMLDivElement;
  private canvas!: HTMLCanvasElement;
  private empty_message!: HTMLParagraphElement;
  private hover_tooltip!: HTMLDivElement;
  private close_button!: HTMLButtonElement;

  private data!: TechTreeDialogData;
  private nodes: TreeNode[] = [];
  private hovered_node?: TreeNode;
  private pending_icon_loads = new Set<HTMLImageElement>();

  constructor() {
    super();
    this.configureElement('title_heading', 'title');
    this.configureElements('canvas_scroll', 'canvas', 'empty_message', 'hover_tooltip', 'close_button');
  }

  override getHTML(): string {
    return html;
  }

  getData(): TechTreeDialogData {
    return this.data;
  }

  setData(data: TechTreeDialogData, parsed?: boolean) {
    this.data = data;
    this.classList.add(`size-${DialogSize.XXLARGE}`);
    if (!parsed && !this.fully_parsed) {
      return;
    }
    this.close_button.addEventListener('click', () => {
      this.closeDialog();
    });
    this.canvas.addEventListener('mousemove', (e) => this.handleMousemove(e));
    this.canvas.addEventListener('mouseleave', () => this.hideTooltip());
    this.loadTree();
  }

  private async loadTree() {
    const response = await apiGet<RisqBuildingTreeEntry[]>('risq/tech-tree');
    if (!response.success || response.result.length === 0) {
      this.canvas.hidden = true;
      this.empty_message.hidden = false;
      return;
    }
    this.canvas.hidden = false;
    this.empty_message.hidden = true;
    this.renderTree(response.result);
  }

  private nodesPerRow(): number {
    return Math.max(1, Math.floor((COLUMN_W + NODE_GAP) / (NODE_S + NODE_GAP)));
  }

  private renderTree(buildings: RisqBuildingTreeEntry[]) {
    const researched_techs = this.data.risq.getPlayer()?.researched_techs;
    const nodes_per_row = this.nodesPerRow();
    this.nodes = [];
    let x = PADDING;
    let max_column_h = 0;
    for (const building of buildings) {
      const producibles = [...building.produces].sort((a, b) => a.row - b.row || a.id - b.id);
      const header_status = techStatus(building.required_tech_id, false, researched_techs);
      this.nodes.push({
        kind: 'header',
        x,
        y: PADDING,
        w: COLUMN_W,
        h: HEADER_H,
        status: header_status === 'researched' ? 'available' : header_status,
        building,
      });
      const rows = Math.ceil(producibles.length / nodes_per_row);
      const nodes_top = PADDING + HEADER_H + HEADER_TO_NODES_GAP;
      producibles.forEach((entry, i) => {
        const row = Math.floor(i / nodes_per_row);
        const col = i % nodes_per_row;
        const researched = entry.kind === RisqProducibleKind.TECH && !!researched_techs?.get(entry.id);
        this.nodes.push({
          kind: 'producible',
          x: x + col * (NODE_S + NODE_GAP),
          y: nodes_top + row * (NODE_S + NODE_GAP),
          w: NODE_S,
          h: NODE_S,
          status: techStatus(entry.required_tech_id, researched, researched_techs),
          entry,
        });
      });
      const column_h = rows > 0 ? nodes_top - PADDING + rows * NODE_S + (rows - 1) * NODE_GAP : HEADER_H;
      max_column_h = Math.max(max_column_h, column_h);
      x += COLUMN_W + COLUMN_GAP;
    }
    const content_w = x - COLUMN_GAP + PADDING;
    const content_h = max_column_h + 2 * PADDING;
    this.drawCanvas(content_w, content_h);
  }

  private drawCanvas(content_w: number, content_h: number) {
    const dpr = window.devicePixelRatio || 1;
    this.canvas.style.width = `${content_w}px`;
    this.canvas.style.height = `${content_h}px`;
    this.canvas.width = content_w * dpr;
    this.canvas.height = content_h * dpr;
    const ctx = this.canvas.getContext('2d');
    if (!ctx) {
      return;
    }
    ctx.scale(dpr, dpr);
    this.redraw(ctx, content_w, content_h);
  }

  private redraw(ctx: CanvasRenderingContext2D, content_w: number, content_h: number) {
    ctx.fillStyle = 'black';
    ctx.fillRect(0, 0, content_w, content_h);
    for (const node of this.nodes) {
      if (node.kind === 'header') {
        this.drawConnectors(ctx, node);
      }
    }
    for (const node of this.nodes) {
      this.drawNode(ctx, node);
    }
  }

  /** Draws a branch line from the header to each of its own producible nodes, so every node is visibly connected */
  private drawConnectors(ctx: CanvasRenderingContext2D, header: HeaderNode) {
    const from = { x: header.x + 0.5 * header.w, y: header.y + header.h };
    ctx.strokeStyle = 'rgba(200, 190, 175, 0.35)';
    ctx.lineWidth = 1.5;
    for (const node of this.nodes) {
      if (node.kind !== 'producible' || node.x < header.x || node.x >= header.x + header.w) {
        continue;
      }
      ctx.beginPath();
      ctx.moveTo(from.x, from.y);
      ctx.lineTo(node.x + 0.5 * node.w, node.y);
      ctx.stroke();
    }
  }

  /** Returns the cached icon, and schedules a redraw once it finishes loading if it isn't ready yet */
  private trackedIcon(path: string): HTMLImageElement {
    const icon = this.data.risq.getIcon(path);
    if (!icon.complete && !this.pending_icon_loads.has(icon)) {
      this.pending_icon_loads.add(icon);
      icon.addEventListener(
        'load',
        () => {
          this.pending_icon_loads.delete(icon);
          this.redrawFromCache();
        },
        { once: true }
      );
    }
    return icon;
  }

  private drawNode(ctx: CanvasRenderingContext2D, node: TreeNode) {
    ctx.fillStyle = STATUS_FILL[node.status];
    ctx.strokeStyle = node === this.hovered_node ? 'white' : STATUS_STROKE[node.status];
    ctx.lineWidth = node === this.hovered_node ? 2.5 : 1.5;
    ctx.beginPath();
    ctx.roundRect(node.x, node.y, node.w, node.h, 6);
    ctx.fill();
    ctx.stroke();

    if (node.kind === 'header') {
      this.drawHeaderContent(ctx, node);
    } else {
      const icon = this.trackedIcon(nodeIcon(node.entry));
      const icon_s = node.w - 2 * ICON_PADDING_NODE;
      ctx.drawImage(icon, node.x + ICON_PADDING_NODE, node.y + ICON_PADDING_NODE, icon_s, icon_s);
    }
  }

  private drawHeaderContent(ctx: CanvasRenderingContext2D, node: HeaderNode) {
    const icon = this.trackedIcon(buildingImage(node.building.building_id, false, true));
    ctx.drawImage(icon, node.x + 8, node.y + 0.5 * node.h - 0.5 * ICON_S_HEADER, ICON_S_HEADER, ICON_S_HEADER);

    ctx.fillStyle = STATUS_TEXT[node.status];
    ctx.textBaseline = 'top';
    ctx.font = 'bold 14px serif';
    const text_x = node.x + 12 + ICON_S_HEADER;
    const text_w = node.w - 20 - ICON_S_HEADER;
    ctx.fillText(this.truncateText(ctx, node.building.display_name, text_w), text_x, node.y + 10, text_w);

    ctx.font = '11px serif';
    ctx.fillStyle = 'rgba(200, 190, 175, 0.85)';
    const lines = this.wrapText(ctx, node.building.description, text_w, 2);
    lines.forEach((line, i) => ctx.fillText(line, text_x, node.y + 30 + i * 13, text_w));
  }

  private truncateText(ctx: CanvasRenderingContext2D, text: string, max_w: number): string {
    if (ctx.measureText(text).width <= max_w) {
      return text;
    }
    let truncated = text;
    while (truncated.length > 1 && ctx.measureText(`${truncated}…`).width > max_w) {
      truncated = truncated.slice(0, -1);
    }
    return `${truncated}…`;
  }

  private wrapText(ctx: CanvasRenderingContext2D, text: string, max_w: number, max_lines: number): string[] {
    const words = text.split(' ');
    const lines: string[] = [];
    let current = '';
    for (const word of words) {
      const attempt = current ? `${current} ${word}` : word;
      if (current && ctx.measureText(attempt).width > max_w) {
        lines.push(current);
        current = word;
        if (lines.length === max_lines - 1) {
          break;
        }
      } else {
        current = attempt;
      }
    }
    if (current && lines.length < max_lines) {
      lines.push(current);
    }
    if (lines.length === max_lines && ctx.measureText(lines[max_lines - 1]).width > max_w) {
      lines[max_lines - 1] = this.truncateText(ctx, lines[max_lines - 1], max_w);
    }
    return lines;
  }

  private handleMousemove(e: MouseEvent) {
    const rect = this.canvas.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;
    const found = this.nodes.find((node) => x >= node.x && x <= node.x + node.w && y >= node.y && y <= node.y + node.h);
    if (found === this.hovered_node) {
      if (found) {
        this.positionTooltip(e);
      }
      return;
    }
    this.hovered_node = found;
    this.redrawFromCache();
    if (found) {
      this.hover_tooltip.replaceChildren(this.buildTooltipContent(found));
      this.hover_tooltip.hidden = false;
      this.positionTooltip(e);
    } else {
      this.hideTooltip();
    }
  }

  private addLine(container: DocumentFragment, text: string, cls = 'tooltip-line') {
    const line = document.createElement('span');
    line.classList.add(cls);
    line.innerText = text;
    container.appendChild(line);
  }

  private addStatChip(container: HTMLElement, icon_path: string, value: number) {
    const chip = document.createElement('span');
    const img = document.createElement('img');
    img.src = `/images/${icon_path}.png`;
    img.alt = '';
    const text = document.createElement('span');
    text.innerText = value.toString();
    chip.append(img, text);
    container.appendChild(chip);
  }

  private addCostRow(fragment: DocumentFragment, cost: RisqCost) {
    const cost_row = document.createElement('div');
    cost_row.classList.add('tooltip-stats');
    const costs: [RisqResourceType, number][] = [
      [RisqResourceType.FOOD, cost.food],
      [RisqResourceType.WOOD, cost.wood],
      [RisqResourceType.STONE, cost.stone],
      [RisqResourceType.GOLD, cost.gold],
    ];
    for (const [resource_type, amount] of costs) {
      if (amount) {
        this.addStatChip(cost_row, resourceTypeImage(resource_type), amount);
      }
    }
    if (cost_row.children.length > 0) {
      fragment.appendChild(cost_row);
    }
  }

  private addStatsRows(fragment: DocumentFragment, stats: RisqUnitStatsEntry) {
    this.addLine(fragment, `Health: ${stats.health}`);
    const combat_row = document.createElement('div');
    combat_row.classList.add('tooltip-stats');
    if (stats.attack_blunt) {
      this.addStatChip(combat_row, 'risq/icons/attack_blunt', stats.attack_blunt);
    }
    if (stats.attack_piercing) {
      this.addStatChip(combat_row, 'risq/icons/attack_piercing', stats.attack_piercing);
    }
    const range_distance = RANGE_DISTANCE[stats.attack_range];
    if (range_distance !== undefined) {
      this.addStatChip(combat_row, 'risq/icons/attack_range', range_distance);
    }
    if (stats.defense_blunt) {
      this.addStatChip(combat_row, 'risq/icons/defense_blunt', stats.defense_blunt);
    }
    if (stats.defense_piercing) {
      this.addStatChip(combat_row, 'risq/icons/defense_piercing', stats.defense_piercing);
    }
    if (stats.penetration_blunt) {
      this.addStatChip(combat_row, 'risq/icons/penetration_blunt', stats.penetration_blunt);
    }
    if (stats.penetration_piercing) {
      this.addStatChip(combat_row, 'risq/icons/penetration_piercing', stats.penetration_piercing);
    }
    if (combat_row.children.length > 0) {
      fragment.appendChild(combat_row);
    }
  }

  private buildTooltipContent(node: TreeNode): DocumentFragment {
    const fragment = document.createDocumentFragment();
    if (node.kind === 'header') {
      const title = document.createElement('span');
      title.classList.add('tooltip-title');
      title.innerText = node.building.display_name;
      const description = document.createElement('span');
      description.classList.add('tooltip-description');
      description.innerText = node.building.description;
      fragment.append(title, description);
      this.addCostRow(fragment, node.building.cost);
      if (node.building.stats) {
        this.addStatsRows(fragment, node.building.stats);
      }
      return fragment;
    }
    const entry = node.entry;
    const title = document.createElement('span');
    title.classList.add('tooltip-title');
    title.innerText = entry.display_name;
    const description = document.createElement('span');
    description.classList.add('tooltip-description');
    description.innerText = entry.description;
    fragment.append(title, description);
    this.addCostRow(fragment, entry.cost);
    if (entry.stats) {
      this.addStatsRows(fragment, entry.stats);
    }

    if (entry.bonus) {
      const affected = (entry.affects_unit_types ?? []).map((t) => RisqUnitType[t]?.toLowerCase() ?? t.toString());
      if (affected.length > 0) {
        this.addLine(fragment, `Affects: ${affected.join(', ')}`);
      } else if ((entry.affects_unit_ids ?? []).length > 0) {
        this.addLine(fragment, `Affects: ${entry.affects_unit_ids!.length} unit(s)`);
      }
      if (entry.bonus.max_health) {
        this.addLine(fragment, `Bonus health: +${entry.bonus.max_health}`);
      }
      if (entry.bonus.turn_stamina) {
        this.addLine(fragment, `Bonus stamina/turn: +${entry.bonus.turn_stamina}`);
      }
      const bonus_row = document.createElement('div');
      bonus_row.classList.add('tooltip-stats');
      const bonus_chips: [string, number][] = [
        ['risq/icons/attack_blunt', entry.bonus.attack_blunt],
        ['risq/icons/attack_piercing', entry.bonus.attack_piercing],
        ['risq/icons/attack_magic', entry.bonus.attack_magic],
        ['risq/icons/defense_blunt', entry.bonus.defense_blunt],
        ['risq/icons/defense_piercing', entry.bonus.defense_piercing],
        ['risq/icons/defense_magic', entry.bonus.defense_magic],
        ['risq/icons/penetration_blunt', entry.bonus.penetration_blunt],
        ['risq/icons/penetration_piercing', entry.bonus.penetration_piercing],
        ['risq/icons/penetration_magic', entry.bonus.penetration_magic],
      ];
      for (const [icon_path, value] of bonus_chips) {
        if (value) {
          this.addStatChip(bonus_row, icon_path, value);
        }
      }
      if (bonus_row.children.length > 0) {
        fragment.appendChild(bonus_row);
      }
    }
    return fragment;
  }

  // flips left/right and up/down based on which half of the visible viewport the cursor is in, so the tooltip is never behind it
  private positionTooltip(e: MouseEvent) {
    const wrapper_rect = this.canvas_scroll.getBoundingClientRect();
    const tooltip_rect = this.hover_tooltip.getBoundingClientRect();
    const gap = 16;
    const cursor_x = e.clientX - wrapper_rect.left;
    const cursor_y = e.clientY - wrapper_rect.top;
    const local_x = cursor_x + this.canvas_scroll.scrollLeft;
    const local_y = cursor_y + this.canvas_scroll.scrollTop;
    const x = cursor_x > wrapper_rect.width / 2 ? local_x - tooltip_rect.width - gap : local_x + gap;
    const y = cursor_y > wrapper_rect.height / 2 ? local_y - tooltip_rect.height - gap : local_y + gap;
    this.hover_tooltip.style.left = `${Math.max(0, x)}px`;
    this.hover_tooltip.style.top = `${Math.max(0, y)}px`;
  }

  private hideTooltip() {
    this.hovered_node = undefined;
    this.hover_tooltip.hidden = true;
    this.redrawFromCache();
  }

  private redrawFromCache() {
    const ctx = this.canvas.getContext('2d');
    if (!ctx) {
      return;
    }
    const dpr = window.devicePixelRatio || 1;
    this.redraw(ctx, this.canvas.width / dpr, this.canvas.height / dpr);
  }
}

customElements.define('dwg-risq-tech-tree-dialog', DwgRisqTechTreeDialog);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-risq-tech-tree-dialog': DwgRisqTechTreeDialog;
  }
}
