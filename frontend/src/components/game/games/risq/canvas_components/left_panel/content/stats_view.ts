import type { BoardTransformData } from '../../../../../util/canvas_board/canvas_board';
import { createTooltipState, drawTooltip, shouldShowTooltip } from '../../../../../util/canvas_components/tooltip';
import { drawRect, drawText } from '../../../../../util/canvas_util';
import type { Point2D } from '../../../../../util/objects2d';
import type { RectHoverData, RisqBuilding, RisqCombatStats, RisqResource, RisqUnit } from '../../../model/types';
import { RisqAttackType, RisqRange, RisqResourceType } from '../../../model/types';
import { resourceTypeImage } from '../../../rendering/assets/resources';
import { UNIT_HEALTHBAR_COLOR_BACKGROUND, UNIT_HEALTHBAR_COLOR_HEALTH } from '../../../rendering/assets/unit';
import type { DwgRisq } from '../../../risq';
import { PANEL_PADDING } from '../layout';
import type { PanelDrawContext } from './primitives';

type StatCell = [icon: string, value: number | string, label: string, tooltip?: string];

const RANGE_LABELS: Record<RisqRange, string> = {
  [RisqRange.NONE]: 'none',
  [RisqRange.ZONE]: 'zone (melee)',
  [RisqRange.SPACE]: 'space',
  [RisqRange.ADJACENT]: '1 space',
  [RisqRange.SECONDARY]: '2 spaces',
};

const HEALTH_ROW_H = 14;
const STAT_ROW_H = 20;

/** "current+delta/capacity" workers at a gatherable target, counting pending gather orders */
export function workersText(risq: DwgRisq, target: RisqBuilding | RisqResource, capacity: number): string {
  const { current, predicted } = risq.planning.gatherers(target);
  const delta = predicted.size - current;
  return `${current}${delta > 0 ? `+${delta}` : delta < 0 ? `-${-delta}` : ''}/${capacity}`;
}

function attackTypeIncludes(attack_type: RisqAttackType, component: 0 | 1 | 2): boolean {
  switch (attack_type) {
    case RisqAttackType.BLUNT:
      return component === 0;
    case RisqAttackType.PIERCING:
      return component === 1;
    case RisqAttackType.MAGIC:
      return component === 2;
    case RisqAttackType.BLUNT_PIERCING:
      return component === 0 || component === 1;
    case RisqAttackType.PIERCING_MAGIC:
      return component === 1 || component === 2;
    case RisqAttackType.MAGIC_BLUNT:
      return component === 2 || component === 0;
    case RisqAttackType.BLUNT_PIERCING_MAGIC:
      return true;
    default:
      return false;
  }
}

// Returns the attack's space radius (0/1/2), or null if this isn't a ranged attack (RisqRange_ZONE)
function rangeDistance(range: RisqRange): number | null {
  switch (range) {
    case RisqRange.SPACE:
      return 0;
    case RisqRange.ADJACENT:
      return 1;
    case RisqRange.SECONDARY:
      return 2;
    default:
      return null;
  }
}

function combatStatsGroups(cs: RisqCombatStats, attack_range: RisqRange): (StatCell | null)[][] {
  const has_attack = cs.attack_type !== RisqAttackType.NONE;
  const includes = (component: 0 | 1 | 2): boolean => attackTypeIncludes(cs.attack_type, component);
  const range = rangeDistance(attack_range);
  const attack: (StatCell | null)[] = has_attack
    ? [
        includes(0) ? ['risq/icons/attack_blunt', cs.attack_blunt, 'Blunt attack'] : null,
        includes(1) ? ['risq/icons/attack_piercing', cs.attack_piercing, 'Piercing attack'] : null,
        includes(2) ? ['risq/icons/attack_magic', cs.attack_magic, 'Magic attack'] : null,
        range !== null
          ? ['risq/icons/attack_range', range, 'Attack range', `Attack range: ${RANGE_LABELS[attack_range]}`]
          : null,
      ]
    : [null, null, null, null];
  const penetration: (StatCell | null)[] = has_attack
    ? [
        cs.penetration_blunt ? ['risq/icons/penetration_blunt', cs.penetration_blunt, 'Blunt penetration'] : null,
        cs.penetration_piercing
          ? ['risq/icons/penetration_piercing', cs.penetration_piercing, 'Piercing penetration']
          : null,
        cs.penetration_magic ? ['risq/icons/penetration_magic', cs.penetration_magic, 'Magic penetration'] : null,
        null,
      ]
    : [null, null, null, null];
  const defense: (StatCell | null)[] = [
    cs.defense_blunt ? ['risq/icons/defense_blunt', cs.defense_blunt, 'Blunt defense'] : null,
    cs.defense_piercing ? ['risq/icons/defense_piercing', cs.defense_piercing, 'Piercing defense'] : null,
    cs.defense_magic ? ['risq/icons/defense_magic', cs.defense_magic, 'Magic defense'] : null,
    null,
  ];
  return [attack, defense, penetration];
}

function gatherStatsRow(risq: DwgRisq, building: RisqBuilding): (StatCell | null)[] | undefined {
  if (building.under_construction || building.gather_capacity === undefined) {
    return undefined;
  }
  return [
    [
      resourceTypeImage(building.resource_category ?? RisqResourceType.ERROR),
      Math.round(building.resources_left ?? 0),
      'Resources left',
    ],
    ['icons/gather32', building.base_gather_speed ?? 0, 'Gather rate'],
    ['icons/villager64', workersText(risq, building, building.gather_capacity), 'Workers'],
    building.renewing && building.renew_stamina !== undefined
      ? [
          'icons/wheat32',
          `${building.renew_stamina - (building.renew_stamina_remaining ?? building.renew_stamina)}/${building.renew_stamina}`,
          'Renewal progress',
        ]
      : null,
  ];
}

/** Health bar, stamina, and per-damage-type stat rows for a unit or building, with their hover tooltips */
export class RisqStatsView {
  private healthbar_row: RectHoverData = { ps: { x: 0, y: 0 }, pe: { x: 0, y: 0 } };
  private stamina_row: RectHoverData = { ps: { x: 0, y: 0 }, pe: { x: 0, y: 0 } };
  private stat_cells: { rect: RectHoverData; text: string }[] = [];
  private hovered_stat_cell = -1;
  private healthbar_tooltip = createTooltipState();
  private stamina_tooltip = createTooltipState();
  private stat_tooltip = createTooltipState();

  /** Height of the stats section, including gathering, construction, and renewal details */
  static height(risq: DwgRisq, building?: RisqBuilding): number {
    const stat_rows = 3 + (building && (building.under_construction || gatherStatsRow(risq, building)) ? 1 : 0);
    const rows = 2 + stat_rows;
    return 2 * HEALTH_ROW_H + stat_rows * STAT_ROW_H + (rows - 1) * PANEL_PADDING;
  }

  clearHover(): void {
    this.healthbar_row.hovered = false;
    this.stamina_row.hovered = false;
  }

  draw(pc: PanelDrawContext, start_yi: number, subject: RisqUnit | RisqBuilding, building?: RisqBuilding): void {
    const { ctx, frame } = pc;
    const cs = subject.combat_stats;
    const health_h = HEALTH_ROW_H;
    const gap = PANEL_PADDING;
    const xi = frame.xi() + 0.1 * frame.w();
    let y = start_yi;

    ctx.strokeStyle = UNIT_HEALTHBAR_COLOR_BACKGROUND;
    ctx.lineWidth = 0.4;
    ctx.fillStyle = UNIT_HEALTHBAR_COLOR_BACKGROUND;
    drawRect(ctx, { x: xi, y }, 0.8 * frame.w(), health_h);
    if (cs.max_health > 0 && cs.health > 0) {
      ctx.fillStyle = UNIT_HEALTHBAR_COLOR_HEALTH;
      drawRect(ctx, { x: xi, y }, (cs.health / cs.max_health) * 0.8 * frame.w(), health_h);
    }
    this.healthbar_row.ps = { x: xi, y };
    this.healthbar_row.pe = { x: xi + 0.8 * frame.w(), y: y + health_h };
    y += health_h + gap;

    const text_y = y + 0.5 * health_h;
    drawText(ctx, `${Math.round(cs.health)} / ${cs.max_health}`, {
      p: { x: xi, y: text_y },
      w: 0.8 * frame.w(),
      fill_style: 'black',
      align: 'left',
      baseline: 'middle',
      font: `${health_h}px serif`,
    });
    const row_right = xi + 0.8 * frame.w();
    const stamina_text = `${subject.current_stamina}`;
    ctx.font = `${health_h}px serif`;
    const stamina_text_width = ctx.measureText(stamina_text).width;
    const stamina_icon_x = row_right - stamina_text_width - health_h - 4;
    ctx.drawImage(pc.risq.getIcon('risq/icons/stamina'), stamina_icon_x, y, health_h, health_h);
    this.stamina_row.ps = { x: stamina_icon_x, y };
    this.stamina_row.pe = { x: row_right, y: y + health_h };
    drawText(ctx, stamina_text, {
      p: { x: row_right, y: text_y },
      w: stamina_text_width,
      fill_style: 'black',
      align: 'right',
      baseline: 'middle',
      font: `${health_h}px serif`,
    });
    y += health_h + gap;

    this.stat_cells = [];
    for (const group of combatStatsGroups(cs, subject.attack_range)) {
      this.drawStatRow(pc, group, y);
      y += STAT_ROW_H + gap;
    }
    const gather_row = building ? gatherStatsRow(pc.risq, building) : undefined;
    if (gather_row) {
      this.drawStatRow(pc, gather_row, y);
      y += STAT_ROW_H + gap;
    }
    if (building?.under_construction) {
      const completed = building.construction_stamina_total - building.stamina_remaining;
      drawText(ctx, `Construction progress: ${completed} / ${building.construction_stamina_total}`, {
        p: { x: xi, y: y + 0.5 * STAT_ROW_H },
        w: 0.9 * frame.w(),
        fill_style: 'black',
        baseline: 'middle',
        font: '18px serif',
      });
    }
  }

  private drawStatRow(pc: PanelDrawContext, row: (StatCell | null)[], y: number): void {
    const dx = (0.8 * pc.frame.w() - 3 * PANEL_PADDING) / 4;
    const image_size = STAT_ROW_H;
    for (const [col, entry] of row.entries()) {
      if (!entry) {
        continue;
      }
      const sx = pc.frame.xi() + 0.1 * pc.frame.w() + col * (dx + PANEL_PADDING);
      this.stat_cells.push({
        rect: { ps: { x: sx, y }, pe: { x: sx + dx, y: y + image_size } },
        text: entry[3] ?? `${entry[2]}: ${entry[1]}`,
      });
      pc.ctx.drawImage(pc.risq.getIcon(entry[0]), sx, y, image_size, image_size);
      drawText(pc.ctx, entry[1].toString(), {
        p: { x: sx + image_size, y: y + 0.5 * image_size },
        w: dx - image_size,
        fill_style: 'black',
        align: 'left',
        baseline: 'middle',
        font: `${0.9 * image_size}px serif`,
      });
    }
  }

  /** Tooltips for whichever stats row is hovered; subject is undefined when no stats are showing */
  drawTooltips(
    ctx: CanvasRenderingContext2D,
    transform: BoardTransformData,
    risq: DwgRisq,
    dt: number,
    subject: RisqUnit | RisqBuilding | undefined
  ): void {
    const canvas_size = risq.viewport.canvasSize();
    if (
      subject &&
      shouldShowTooltip(this.healthbar_tooltip, !!this.healthbar_row.hovered, !!this.healthbar_row.clicked, dt)
    ) {
      const cs = subject.combat_stats;
      drawTooltip(
        this.healthbar_tooltip,
        ctx,
        transform,
        canvas_size,
        `Health: ${cs.health.toFixed(1)} / ${cs.max_health}`
      );
    }
    if (
      subject &&
      shouldShowTooltip(this.stamina_tooltip, !!this.stamina_row.hovered, !!this.stamina_row.clicked, dt)
    ) {
      drawTooltip(this.stamina_tooltip, ctx, transform, canvas_size, `Stamina: +${subject.turn_stamina} / turn`);
    }
    const stat_cell = subject ? this.stat_cells[this.hovered_stat_cell] : undefined;
    if (shouldShowTooltip(this.stat_tooltip, !!stat_cell, false, dt) && stat_cell) {
      drawTooltip(this.stat_tooltip, ctx, transform, canvas_size, stat_cell.text);
    }
  }

  mousemove(m: Point2D): void {
    rectHovered(m, this.healthbar_row);
    rectHovered(m, this.stamina_row);
    this.hovered_stat_cell = this.stat_cells.findIndex((cell) => rectHovered(m, cell.rect));
  }
}

/** Sets and returns whether m lies inside the rect */
export function rectHovered(m: Point2D, hover_data: RectHoverData): boolean {
  if (m.x < hover_data.ps.x || m.y < hover_data.ps.y || m.x > hover_data.pe.x || m.y > hover_data.pe.y) {
    hover_data.hovered = false;
    return false;
  }
  hover_data.hovered = true;
  return true;
}
