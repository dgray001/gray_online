import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawRect, drawText } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import { axialDistance, equalsPoint2D } from '../../../../util/objects2d';
import { buildingImage } from '../../risq_buildings';
import type { RisqFrontendOrder } from '../../risq_data';
import {
  coordinateToIndex,
  getSpace,
  invertBuildKey,
  invertPair,
  invertZoneKey,
  isPlayerOrder,
  RisqOrderType,
  RisqResourceType,
} from '../../risq_data';
import { resourceTypeImage } from '../../risq_resources';
import { unitImage } from '../../risq_unit';
import type { RisqOrderRowConfig } from './order_row_data';

const MOVE_INTRA_STAMINA_COST = 1;
const MOVE_INTER_STAMINA_COST = 6;
const GATHER_TICK_STAMINA_COST = 3;

const ROW_H = 30;
const COLLAPSED_STRIP_H = 14;
const ICON_S = 20;
const PADDING = 5;
const CANCEL_S = 14;
const CHIP_W = 30;

interface CostChip {
  icon: string;
  amount: number;
}

interface ResolvedRow {
  icon: string;
  name: string;
  target: string;
  cost: CostChip[];
  subject_icon?: string;
}

export class RisqOrderRow implements CanvasComponent {
  private config: RisqOrderRowConfig;
  private hovering = false;
  private clicking = false;
  private cancel_hover = false;
  private cancel_clicking = false;
  private cancel_all_hover = false;
  private cancel_all_clicking = false;
  private resolved: ResolvedRow;

  constructor(config: RisqOrderRowConfig) {
    this.config = config;
    this.resolved = this.resolve();
  }

  getOrder(): RisqFrontendOrder {
    return this.config.order;
  }

  private resolve(): ResolvedRow {
    const order = this.config.order;
    const game = this.config.game.getGame();
    const player = game?.players[order.player_id];
    const subject_unit = player?.units.get(order.subjects[0]);
    const subject_building = player?.buildings.get(order.subjects[0]);
    const board_size = game?.board_size ?? 0;

    const distance_text = (target_space: Point2D, target_zone?: Point2D): string => {
      if (!subject_unit) {
        return '';
      }
      if (!equalsPoint2D(subject_unit.space_coordinate, target_space)) {
        const d = axialDistance(subject_unit.space_coordinate, target_space);
        return `${d} space${d === 1 ? '' : 's'} away`;
      }
      if (!target_zone) {
        return 'here';
      }
      const d = axialDistance(subject_unit.zone_coordinate, target_zone);
      return d === 0 ? 'here' : `${d} zone${d === 1 ? '' : 's'} away`;
    };

    const move_cost = (target_space: Point2D, target_zone?: Point2D): CostChip[] => {
      if (!subject_unit) {
        return [];
      }
      let amount = 0;
      if (!equalsPoint2D(subject_unit.space_coordinate, target_space)) {
        amount = axialDistance(subject_unit.space_coordinate, target_space) * MOVE_INTER_STAMINA_COST;
      } else if (target_zone) {
        amount = axialDistance(subject_unit.zone_coordinate, target_zone) * MOVE_INTRA_STAMINA_COST;
      }
      return amount > 0 ? [{ icon: 'risq/icons/stamina', amount }] : [];
    };

    const resource_cost = (
      cost: { food: number; wood: number; stone: number } | undefined,
      multiplier = 1
    ): CostChip[] => {
      if (!cost) {
        return [];
      }
      const chips: CostChip[] = [];
      if (cost.wood) {
        chips.push({ icon: resourceTypeImage(RisqResourceType.WOOD), amount: cost.wood * multiplier });
      }
      if (cost.food) {
        chips.push({ icon: resourceTypeImage(RisqResourceType.FOOD), amount: cost.food * multiplier });
      }
      if (cost.stone) {
        chips.push({ icon: resourceTypeImage(RisqResourceType.STONE), amount: cost.stone * multiplier });
      }
      return chips;
    };

    const zone_resource_name = (space: Point2D, zone: Point2D): string => {
      if (!game) {
        return 'a resource';
      }
      const found = getSpace(game, coordinateToIndex(board_size, space));
      const zone_data = found?.zones?.flat().find((z) => equalsPoint2D(z.coordinate, zone));
      return zone_data?.resource?.display_name ?? 'a resource';
    };

    let base: { icon: string; name: string; target: string; cost: CostChip[] };
    switch (order.order_type) {
      case RisqOrderType.OrderType_UnitMoveSpace: {
        const target_space = invertPair(order.target_id);
        base = { icon: 'icons/move', name: 'Move', target: distance_text(target_space), cost: move_cost(target_space) };
        break;
      }
      case RisqOrderType.OrderType_UnitMoveZone: {
        const { space, zone } = invertZoneKey(order.target_id);
        base = { icon: 'icons/move', name: 'Move', target: distance_text(space, zone), cost: move_cost(space, zone) };
        break;
      }
      case RisqOrderType.OrderType_UnitGather: {
        const { space, zone } = invertZoneKey(order.target_id);
        base = {
          icon: 'icons/gather',
          name: 'Gather',
          target: zone_resource_name(space, zone),
          cost: [{ icon: 'risq/icons/stamina', amount: GATHER_TICK_STAMINA_COST }],
        };
        break;
      }
      case RisqOrderType.OrderType_UnitBuild: {
        const { building_id, space, zone } = invertBuildKey(order.target_id);
        const producible = subject_unit?.builds.find((p) => p.id === building_id);
        base = {
          icon: buildingImage(building_id),
          name: `Build ${producible?.display_name ?? 'Building'}`,
          target: distance_text(space, zone),
          cost: resource_cost(producible?.cost),
        };
        break;
      }
      case RisqOrderType.OrderType_BuildingCreate: {
        const producible = subject_building?.produces.find((p) => p.id === order.target_id);
        const qty = 1 + (this.config.collapsed_orders?.length ?? 0);
        base = {
          icon: unitImage(order.target_id),
          name:
            qty > 1
              ? `Create ${producible?.display_name ?? 'Unit'} ×${qty}`
              : `Create ${producible?.display_name ?? 'Unit'}`,
          target: '',
          cost: resource_cost(producible?.cost, qty),
        };
        break;
      }
      case RisqOrderType.OrderType_CancelOrder: {
        const cancelled = player?.active_orders.find((o) => o.internal_id === order.target_id);
        base = {
          icon: 'icons/close_gray32',
          name: 'Cancel',
          target: cancelled ? this.shortLabel(cancelled.order_type) : 'an order',
          cost: [],
        };
        break;
      }
      case RisqOrderType.OrderType_CancelFoundation: {
        const { space, zone } = invertZoneKey(order.target_id);
        base = { icon: 'icons/close_gray32', name: 'Cancel Foundation', target: distance_text(space, zone), cost: [] };
        break;
      }
      default:
        base = { icon: 'icons/fist32', name: this.shortLabel(order.order_type), target: '', cost: [] };
        break;
    }

    if (isPlayerOrder(order.order_type)) {
      return base;
    }
    return {
      ...base,
      subject_icon: subject_unit
        ? unitImage(subject_unit.unit_id)
        : subject_building
          ? buildingImage(subject_building.building_id)
          : undefined,
    };
  }

  private shortLabel(order_type: RisqOrderType): string {
    switch (order_type) {
      case RisqOrderType.OrderType_UnitMoveSpace:
      case RisqOrderType.OrderType_UnitMoveZone:
        return 'Move';
      case RisqOrderType.OrderType_UnitGather:
        return 'Gather';
      case RisqOrderType.OrderType_UnitBuild:
        return 'Build';
      case RisqOrderType.OrderType_BuildingCreate:
        return 'Create';
      case RisqOrderType.OrderType_UnitDelete:
        return 'Delete';
      default:
        return 'Order';
    }
  }

  isHovering(): boolean {
    return this.hovering;
  }
  setHovering(hovering: boolean): void {
    this.hovering = hovering;
  }
  isClicking(): boolean {
    return this.clicking;
  }
  setClicking(clicking: boolean): void {
    this.clicking = clicking;
  }

  setW(w: number): void {
    this.config.w = w;
  }

  private isCollapsed(): boolean {
    return !!this.config.collapsed_orders?.length;
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, _dt: number): void {
    configDraw(
      ctx,
      transform,
      {
        fill_style: 'rgb(241, 226, 196)',
        stroke_style: 'rgb(59, 36, 19)',
        stroke_width: 0.6,
        hover_fill_style: 'rgb(247, 236, 212)',
        click_fill_style: 'rgb(252, 244, 224)',
      },
      this.isHovering(),
      this.isClicking(),
      () => {
        drawRect(ctx, { x: this.xi(), y: this.yi() }, this.w(), ROW_H, 3);
        let x = this.xi() + PADDING;
        const yc = this.yi() + ROW_H / 2;
        ctx.drawImage(this.config.game.getIcon(this.resolved.icon), x, yc - ICON_S / 2, ICON_S, ICON_S);
        x += ICON_S + 5;
        if (this.config.show_subject && this.resolved.subject_icon) {
          ctx.drawImage(this.config.game.getIcon(this.resolved.subject_icon), x, yc - ICON_S / 2, ICON_S, ICON_S);
          const count = this.config.order.subjects.length;
          if (count > 1) {
            drawText(ctx, `×${count}`, {
              p: { x, y: yc + ICON_S / 2 - 7 },
              w: ICON_S,
              fill_style: 'rgb(59, 36, 19)',
              align: 'right',
              font: 'bold 8px sans-serif',
            });
          }
          x += ICON_S + 5;
        }
        const chips_w = this.chipsWidth();
        const text_w = this.w() - (x - this.xi()) - chips_w - PADDING - CANCEL_S;
        drawText(ctx, this.resolved.name, {
          p: { x, y: yc - 8 },
          w: text_w,
          fill_style: 'rgb(59, 36, 19)',
          align: 'left',
          font: 'bold 10.5px serif',
        });
        drawText(ctx, this.resolved.target, {
          p: { x, y: yc + 3 },
          w: text_w,
          fill_style: 'rgb(122, 92, 62)',
          align: 'left',
          font: '9.5px sans-serif',
        });
        this.drawChips(ctx, this.xi() + this.w() - PADDING - CANCEL_S - chips_w, yc);
        this.drawCancel(ctx, yc);
        if (this.isCollapsed()) {
          this.drawCancelAllStrip(ctx);
        }
      }
    );
  }

  private chipsWidth(): number {
    return this.resolved.cost.length * (CHIP_W + 3);
  }

  private drawChips(ctx: CanvasRenderingContext2D, x: number, yc: number) {
    for (const chip of this.resolved.cost) {
      ctx.drawImage(this.config.game.getIcon(chip.icon), x, yc - 7, 14, 14);
      drawText(ctx, chip.amount.toString(), {
        p: { x: x + 16, y: yc },
        w: CHIP_W - 16,
        fill_style: 'rgb(59, 36, 19)',
        align: 'left',
        baseline: 'middle',
        font: '10px sans-serif',
      });
      x += CHIP_W + 3;
    }
  }

  private drawCancel(ctx: CanvasRenderingContext2D, yc: number) {
    const x = this.xi() + this.w() - PADDING - CANCEL_S;
    const y = yc - CANCEL_S / 2;
    ctx.fillStyle = this.cancel_clicking
      ? 'rgb(156, 60, 37)'
      : this.cancel_hover
        ? 'rgba(122, 46, 27, 0.85)'
        : 'transparent';
    ctx.strokeStyle = 'transparent';
    drawRect(ctx, { x, y }, CANCEL_S, CANCEL_S, 2);
    drawText(ctx, '×', {
      p: { x, y: yc },
      w: CANCEL_S,
      fill_style: this.cancel_hover ? 'white' : 'rgb(122, 92, 62)',
      align: 'center',
      baseline: 'middle',
      font: 'bold 11px sans-serif',
    });
  }

  private drawCancelAllStrip(ctx: CanvasRenderingContext2D) {
    const y = this.yi() + ROW_H;
    ctx.fillStyle = this.cancel_all_clicking
      ? 'rgba(122, 46, 27, 0.35)'
      : this.cancel_all_hover
        ? 'rgba(122, 46, 27, 0.2)'
        : 'rgba(59, 36, 19, 0.06)';
    ctx.strokeStyle = 'transparent';
    drawRect(ctx, { x: this.xi(), y }, this.w(), COLLAPSED_STRIP_H, 0);
    const count = 1 + (this.config.collapsed_orders?.length ?? 0);
    drawText(ctx, `Cancel all ×${count}`, {
      p: { x: this.xi(), y: y + COLLAPSED_STRIP_H / 2 },
      w: this.w(),
      fill_style: 'rgb(122, 46, 27)',
      align: 'center',
      baseline: 'middle',
      font: '9px sans-serif',
    });
  }

  scroll(_dy: number, _mode: number): boolean {
    return false;
  }

  mousemove(m: Point2D, transform: BoardTransformData): boolean {
    m = {
      x: m.x * transform.scale - transform.view.x,
      y: m.y * transform.scale - transform.view.y,
    };
    this.hovering = !(m.x < this.xi() || m.y < this.yi() || m.x > this.xf() || m.y > this.yf());
    const cx = this.xi() + this.w() - PADDING - CANCEL_S;
    const cy = this.yi() + ROW_H / 2 - CANCEL_S / 2;
    this.cancel_hover = this.hovering && m.x >= cx && m.x <= cx + CANCEL_S && m.y >= cy && m.y <= cy + CANCEL_S;
    if (this.isCollapsed()) {
      const sy = this.yi() + ROW_H;
      this.cancel_all_hover = this.hovering && m.y >= sy && m.y <= sy + COLLAPSED_STRIP_H;
    } else {
      this.cancel_all_hover = false;
    }
    return this.hovering;
  }

  mousedown(_e: MouseEvent): boolean {
    if (this.cancel_hover) {
      this.cancel_clicking = true;
      return true;
    }
    if (this.cancel_all_hover) {
      this.cancel_all_clicking = true;
      return true;
    }
    if (this.hovering) {
      this.clicking = true;
      return true;
    }
    return false;
  }

  mouseup(_e: MouseEvent): void {
    if (this.cancel_clicking && this.cancel_hover) {
      const collapsed = this.config.collapsed_orders;
      this.config.onCancel(collapsed?.length ? collapsed[collapsed.length - 1] : this.config.order);
    }
    if (this.cancel_all_clicking && this.cancel_all_hover && this.config.collapsed_orders) {
      this.config.onCancelAll?.([this.config.order, ...this.config.collapsed_orders]);
    }
    this.cancel_clicking = false;
    this.cancel_all_clicking = false;
    this.clicking = false;
  }

  xi(): number {
    return 0;
  }
  yi(): number {
    return 0;
  }
  xf(): number {
    return this.w();
  }
  yf(): number {
    return this.h();
  }
  w(): number {
    return this.config.w;
  }
  h(): number {
    return this.isCollapsed() ? ROW_H + COLLAPSED_STRIP_H : ROW_H;
  }
}
