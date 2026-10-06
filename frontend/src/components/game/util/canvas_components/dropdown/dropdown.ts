import type { BoardTransformData } from '../../canvas_board/canvas_board';
import type { Point2D } from '../../objects2d';
import type { CanvasComponent } from '../canvas_component';

export declare interface DropdownOption {
  value: string;
  label: string;
}

export declare interface DropdownConfig {
  options: DropdownOption[];
  selected?: string;
  start_disabled?: boolean;
}

type DropdownPress = 'button' | number;

export abstract class DwgDropdown implements CanvasComponent {
  private hovering = false;
  private over_popover = false;
  private hover_option?: number;
  private pressed?: DropdownPress;
  private open = false;
  private disabled: boolean;
  private option_list: DropdownOption[];
  private selected?: string;

  constructor(config: DropdownConfig) {
    this.option_list = config.options;
    this.selected = config.selected;
    this.disabled = !!config.start_disabled;
  }

  options(): DropdownOption[] {
    return this.option_list;
  }

  setOptions(options: DropdownOption[]) {
    this.option_list = options;
    if (!options.some((option) => option.value === this.selected)) {
      this.selected = undefined;
    }
    this.close();
  }

  value(): string | undefined {
    return this.selected;
  }

  setValue(value: string | undefined) {
    this.selected = this.option_list.some((option) => option.value === value) ? value : undefined;
  }

  selectedOption(): DropdownOption | undefined {
    return this.option_list.find((option) => option.value === this.selected);
  }

  isOpen(): boolean {
    return this.open;
  }

  close() {
    this.open = false;
    this.over_popover = false;
    this.hover_option = undefined;
    this.pressed = undefined;
  }

  protected hoveredOption(): number | undefined {
    return this.hover_option;
  }

  protected isButtonHovered(): boolean {
    return this.hovering;
  }

  isHovering() {
    return this.hovering || this.over_popover;
  }

  setHovering(hovering: boolean) {
    if (!hovering) {
      this.hovering = false;
      this.over_popover = false;
      this.hover_option = undefined;
    }
  }

  isClicking() {
    return this.pressed !== undefined;
  }

  setClicking(clicking: boolean) {
    if (!clicking) {
      this.pressed = undefined;
    }
  }

  isDisabled(): boolean {
    return this.disabled;
  }

  disable() {
    this.disabled = true;
    this.setHovering(false);
    this.close();
  }

  enable() {
    this.disabled = false;
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    this._draw(ctx, transform, dt);
  }

  mousemove(canvas: Point2D, screen: Point2D, _transform: BoardTransformData): boolean {
    if (this.disabled) {
      this.setHovering(false);
      return false;
    }
    this.hovering = this.mouseOver(canvas, screen);
    this.over_popover = this.open && this.overPopover(canvas, screen);
    this.hover_option = this.over_popover ? this.optionAt(canvas, screen) : undefined;
    return this.isHovering();
  }

  scroll(dy: number): boolean {
    return this.over_popover && this.scrolled(dy);
  }

  mousedown(e: MouseEvent): boolean {
    if (this.disabled || e.button !== 0) {
      return false;
    }
    if (this.hovering) {
      this.pressed = 'button';
      return true;
    }
    if (this.hover_option !== undefined) {
      this.pressed = this.hover_option;
      return true;
    }
    if (this.over_popover) {
      return true;
    }
    const was_open = this.open;
    this.close();
    return was_open;
  }

  mouseup(e: MouseEvent) {
    const target = this.pressed;
    this.pressed = undefined;
    if (e.button !== 0 || target === undefined) {
      return;
    }
    if (target === 'button' && this.hovering) {
      this.toggle();
    } else if (target === this.hover_option && typeof target === 'number') {
      this.choose(target);
    }
  }

  private toggle() {
    if (this.open) {
      this.close();
      return;
    }
    this.open = true;
    this.opened();
  }

  private choose(index: number) {
    const option = this.option_list[index];
    this.close();
    if (option.value !== this.selected) {
      this.selected = option.value;
      this.changed(option);
    }
  }

  protected abstract _draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void;
  abstract mouseOver(canvas: Point2D, screen: Point2D): boolean;
  protected abstract overPopover(canvas: Point2D, screen: Point2D): boolean;
  protected abstract optionAt(canvas: Point2D, screen: Point2D): number | undefined;
  protected abstract scrolled(dy: number): boolean;

  protected abstract opened(): void;
  protected abstract changed(option: DropdownOption): void;

  abstract xi(): number;
  abstract xf(): number;
  abstract yi(): number;
  abstract yf(): number;
  abstract w(): number;
  abstract h(): number;
}
