import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { DwgRectButton } from '../../../../util/canvas_components/button/rect_button';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import type { RisqBuilding, RisqUnit } from '../../model/types';
import { buildingImage } from '../../rendering/assets/buildings';
import { unitImage } from '../../rendering/assets/unit';
import { RisqControlGroupIcon } from './control_group_icon';
import { CONTROL_GAP } from './control_group_layout';
import { IDLE_FILTERS, RisqIdleFilterButton } from './idle_filter_button';
import { clipBottomPanelItem } from './clip_item';

export class RisqIdleGrid extends DwgRectButton {
  private filters: RisqIdleFilterButton[];
  private buttons: RisqControlGroupIcon[] = [];
  private available_width = 0;
  private available_height = 0;
  private content_width = 0;
  private scroll_x = 0;
  private layout_key = '';
  private snapshot: ReturnType<DwgRisq['getGame']>;
  constructor(private risq: DwgRisq) {
    super({
      button_config: {},
      p: { x: 0, y: 0 },
      w: 0,
      h: 0,
      draw_config: { fill_style: 'transparent', stroke_width: 0, fixed_position: true },
    });
    this.filters = IDLE_FILTERS.map(
      ([category, image, description], col): RisqIdleFilterButton =>
        new RisqIdleFilterButton(risq, category, image, description, col)
    );
  }
  setAvailableWidth(width: number): void {
    this.available_width = Math.max(0, width);
    this.dataRefreshed();
  }
  setAvailableHeight(height: number): void {
    this.available_height = height;
    this.dataRefreshed();
  }
  private iconSize(rows: number): number {
    return Math.max(0, (this.available_height - (rows - 1) * CONTROL_GAP) / rows);
  }
  private listOffset(): number {
    return this.iconSize(4) + CONTROL_GAP;
  }
  override setPosition(p: Point2D): void {
    if (p.x === this.xi() && p.y === this.yi()) {
      return;
    }
    super.setPosition(p);
    this.layout_key = '';
    this.dataRefreshed();
  }
  dataRefreshed(): void {
    const entries = this.risq.planning.idleOrderables();
    const key = JSON.stringify([
      this.available_width,
      this.available_height,
      this.xi(),
      this.yi(),
      this.scroll_x,
      entries.map(
        (entity: RisqUnit | RisqBuilding): string => `${'unit_id' in entity ? 'u' : 'b'}${entity.internal_id}`
      ),
    ]);
    const game = this.risq.getGame();
    if (key === this.layout_key && game === this.snapshot) {
      return;
    }
    this.layout_key = key;
    this.snapshot = game;
    const cols = Math.ceil(entries.length / 5);
    const step = this.iconSize(5) + CONTROL_GAP;
    this.content_width = Math.max(0, cols * step - CONTROL_GAP);
    this.setSize(
      this.risq.getPlayer() ? Math.min(this.available_width, this.listOffset() + this.content_width) : 0,
      this.available_height
    );
    this.scroll_x = Math.min(
      this.scroll_x,
      Math.max(0, this.content_width - Math.max(0, this.w() - this.listOffset()))
    );
    this.layoutFilters();
    this.rebuildIcons(entries);
  }
  private layoutFilters(): void {
    const size = this.iconSize(4);
    for (const [i, filter] of this.filters.entries()) {
      filter.setSize(size, size);
      filter.setPosition({ x: this.xi(), y: this.yi() + i * (size + CONTROL_GAP) });
    }
  }
  private rebuildIcons(entries: (RisqUnit | RisqBuilding)[]): void {
    const player = this.risq.getPlayer();
    const size = this.iconSize(5);
    const step = size + CONTROL_GAP;
    this.buttons = player
      ? entries.map((entity: RisqUnit | RisqBuilding, i: number): RisqControlGroupIcon => {
          const kind = 'unit_id' in entity ? 'unit' : 'building';
          const image = 'unit_id' in entity ? unitImage(entity.unit_id) : buildingImage(entity.building_id, false);
          return new RisqControlGroupIcon(
            (): CanvasImageSource => this.risq.getPlayerColoredIcon(image, player.color),
            {
              x: this.xi() + this.listOffset() + Math.floor(i / 5) * step - this.scroll_x,
              y: this.yi() + (i % 5) * step,
            },
            size,
            1,
            { kind, ids: [entity.internal_id] },
            (event: MouseEvent): void => {
              if (event.button === 0) {
                this.risq.selectIdleOrderable(entity);
              }
            }
          );
        })
      : [];
  }
  protected override _draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    ctx.save();
    clipBottomPanelItem(ctx, transform, this);
    for (const filter of this.filters) {
      filter.draw(ctx, transform, dt);
    }
    ctx.save();
    clipBottomPanelItem(ctx, transform, {
      xi: (): number => Math.min(this.xf(), this.xi() + this.listOffset()),
      xf: (): number => this.xf(),
      yi: (): number => this.yi(),
      yf: (): number => this.yf(),
    });
    const units = this.risq.selection.selectedUnitIds();
    const buildings = new Set(this.risq.selection.selectedBuildingIds());
    for (const button of this.buttons) {
      button.setSelection(units, buildings);
      button.draw(ctx, transform, dt);
    }
    ctx.restore();
    ctx.restore();
  }
  drawTooltip(ctx: CanvasRenderingContext2D, transform: BoardTransformData, risq: DwgRisq, dt: number): void {
    for (const filter of this.filters) {
      filter.drawTooltip(ctx, transform, risq, dt);
    }
  }
  override mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    const hovering = super.mousemove(canvas, screen, transform);
    for (const control of [...this.filters, ...this.buttons]) {
      if (hovering && (control instanceof RisqIdleFilterButton || screen.x >= this.xi() + this.listOffset())) {
        control.mousemove(canvas, screen, transform);
      } else {
        control.setHovering(false);
      }
    }
    return hovering;
  }
  override mousedown(e: MouseEvent): boolean {
    return (
      [...this.filters, ...this.buttons].map((control): boolean => control.mousedown(e)).some(Boolean) ||
      this.isHovering()
    );
  }
  override mouseup(e: MouseEvent): void {
    for (const control of [...this.filters, ...this.buttons]) {
      control.mouseup(e);
    }
  }
  override isClicking(): boolean {
    return [...this.filters, ...this.buttons].some((control): boolean => control.isPressed());
  }
  scroll(dy: number, mode: number): boolean {
    const visible_width = Math.max(0, this.w() - this.listOffset());
    const next = Math.max(0, Math.min(this.content_width - visible_width, this.scroll_x + dy * (mode ? 16 : 1)));
    if (next === this.scroll_x) {
      return false;
    }
    this.scroll_x = next;
    this.dataRefreshed();
    return true;
  }
  override setHovering(hovering: boolean): void {
    super.setHovering(hovering);
    if (!hovering) {
      [...this.filters, ...this.buttons].forEach((control): void => control.setHovering(false));
    }
  }
  override setClicking(clicking: boolean): void {
    super.setClicking(clicking);
    if (!clicking) {
      [...this.filters, ...this.buttons].forEach((control): void => control.setClicking(false));
    }
  }
  protected hovered(): void {}
  protected unhovered(): void {}
  protected clicked(): void {}
  protected released(): void {}
}
