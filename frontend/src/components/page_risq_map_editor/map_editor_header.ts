import type { BoardTransformData } from '../game/util/canvas_board/canvas_board';
import type { CanvasComponent, DrawConfig } from '../game/util/canvas_components/canvas_component';
import { configDraw } from '../game/util/canvas_components/canvas_component';
import type { DropdownOption } from '../game/util/canvas_components/dropdown/dropdown';
import { DwgRectDropdown } from '../game/util/canvas_components/dropdown/rect_dropdown';
import { DwgRectInput } from '../game/util/canvas_components/input/rect_input';
import { drawRect, drawText } from '../game/util/canvas_util';
import type { Point2D } from '../game/util/objects2d';
import { EditorButton } from './editor_button';
import { CONTROL_DRAW, CONTROL_H, FONT, HEADER_H, TEXT_CONFIG } from './editor_style';

const CONTROL_Y = 6;
const GAP = 8;
const BAR_FILL = 'rgb(50, 50, 50)';
const INPUT_DRAW: DrawConfig = {
  ...CONTROL_DRAW,
  click_fill_style: 'rgb(255, 250, 235)',
  click_stroke_style: 'rgb(20, 100, 200)',
  click_stroke_width: 1.6,
};
const LABEL_FILL = 'rgb(230, 230, 230)';
const ERROR_FILL = 'rgb(255, 130, 130)';

export interface MapEditorHeaderActions {
  load: (name: string) => void;
  create: (name: string, size: number) => void;
  save: (name: string) => void;
  remove: (name: string) => void;
}

class HeaderInput extends DwgRectInput {
  constructor(
    p: Point2D,
    w: number,
    placeholder: string,
    input_config: { value?: string; max_length: number; allowed_chars: RegExp }
  ) {
    super({
      input_config,
      p,
      w,
      h: CONTROL_H,
      draw_config: INPUT_DRAW,
      text_config: TEXT_CONFIG,
      placeholder,
      placeholder_fill_style: 'rgba(59, 36, 19, 0.5)',
      padding: 6,
      r: 3,
    });
  }

  protected changed(): void {}
  protected submitted(): void {}
}

class HeaderDropdown extends DwgRectDropdown {
  constructor(
    p: Point2D,
    w: number,
    private on_change: (value: string) => void
  ) {
    super({
      dropdown_config: { options: [] },
      p,
      w,
      h: CONTROL_H,
      draw_config: CONTROL_DRAW,
      popover_draw_config: { ...CONTROL_DRAW, hover_fill_style: undefined, stroke_width: 1 },
      text_config: TEXT_CONFIG,
      option_hover_fill_style: 'rgba(59, 36, 19, 0.15)',
      placeholder: 'Select a map...',
      padding: 6,
      r: 3,
    });
  }

  protected changed(option: DropdownOption): void {
    this.on_change(option.value);
  }
}

export class MapEditorHeader implements CanvasComponent {
  private maps: HeaderDropdown;
  private name_input: HeaderInput;
  private size_input: HeaderInput;
  private new_button: EditorButton;
  private save_button: EditorButton;
  private delete_button: EditorButton;
  private size_label_x: number;
  private status_x: number;
  private status = '';
  private status_is_error = false;
  private save_shown = false;
  private delete_shown = false;
  private over_bar = false;

  constructor(actions: MapEditorHeaderActions) {
    let x = GAP;
    const next = (w: number): Point2D => {
      const p = { x, y: CONTROL_Y };
      x += w + GAP;
      return p;
    };
    this.maps = new HeaderDropdown(next(170), 170, (name) => actions.load(name));
    const name_config = { max_length: 40, allowed_chars: /[a-z0-9_]/ };
    this.name_input = new HeaderInput(next(170), 170, 'map_name', name_config);
    this.size_label_x = x;
    x += 32;
    this.size_input = new HeaderInput(next(36), 36, '', { value: '2', max_length: 2, allowed_chars: /[0-9]/ });
    this.new_button = new EditorButton('New', next(52), 52, () => actions.create(this.name(), this.size()));
    this.save_button = new EditorButton('Save', next(52), 52, () => actions.save(this.name()));
    this.delete_button = new EditorButton('Delete', next(60), 60, () => actions.remove(this.name()));
    this.status_x = x;
  }

  private active(): CanvasComponent[] {
    const always = [this.maps, this.name_input, this.size_input, this.new_button];
    const shown = this.save_shown ? [...always, this.save_button] : always;
    return this.delete_shown ? [...shown, this.delete_button] : shown;
  }

  name(): string {
    return this.name_input.value().trim();
  }

  size(): number {
    return Number.parseInt(this.size_input.value());
  }

  isTyping(): boolean {
    return this.name_input.isFocused() || this.size_input.isFocused();
  }

  setName(name: string): void {
    this.name_input.setValue(name);
  }

  mapNames(): string[] {
    return this.maps.options().map((option) => option.value);
  }

  setMaps(names: string[], selected?: string): void {
    this.maps.setOptions(names.map((value) => ({ value, label: value })));
    this.maps.setValue(selected);
  }

  setSelected(name: string | undefined): void {
    this.maps.setValue(name);
  }

  setStatus(text: string, is_error = false): void {
    this.status = text;
    this.status_is_error = is_error;
  }

  showSave(shown: boolean): void {
    this.save_shown = shown;
  }

  showDelete(shown: boolean): void {
    this.delete_shown = shown;
  }

  isHovering(): boolean {
    return this.over_bar || this.active().some((c) => c.isHovering());
  }

  setHovering(hovering: boolean): void {
    this.over_bar = this.over_bar && hovering;
    this.active().forEach((c) => c.setHovering(hovering));
  }

  isClicking(): boolean {
    return this.active().some((c) => c.isClicking());
  }

  setClicking(clicking: boolean): void {
    this.active().forEach((c) => c.setClicking(clicking));
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    const bar = { fill_style: BAR_FILL, stroke_width: 0, fixed_position: true };
    configDraw(ctx, transform, bar, false, false, () => {
      drawRect(ctx, { x: 0, y: 0 }, ctx.canvas.width, HEADER_H);
      const text = { font: FONT, baseline: 'middle' as const, w: 600 };
      drawText(ctx, 'Size', { ...text, p: { x: this.size_label_x, y: HEADER_H / 2 }, fill_style: LABEL_FILL, w: 30 });
      drawText(ctx, this.status, {
        ...text,
        p: { x: this.status_x, y: HEADER_H / 2 },
        fill_style: this.status_is_error ? ERROR_FILL : LABEL_FILL,
      });
    });
    this.active().forEach((c) => c.draw(ctx, transform, dt));
  }

  scroll(dy: number, mode: number, dx?: number): boolean {
    return this.active().some((c) => c.scroll?.(dy, mode, dx));
  }

  mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    this.over_bar = screen.y <= HEADER_H;
    const over_control = this.active().map((c) => c.mousemove(canvas, screen, transform));
    return this.over_bar || over_control.some(Boolean);
  }

  mousedown(e: MouseEvent): boolean {
    const consumed = this.active().map((c) => c.mousedown(e));
    return this.over_bar || consumed.some(Boolean);
  }

  mouseup(e: MouseEvent): void {
    this.active().forEach((c) => c.mouseup(e));
  }

  xi(): number {
    return 0;
  }
  yi(): number {
    return 0;
  }
  xf(): number {
    return this.status_x;
  }
  yf(): number {
    return HEADER_H;
  }
  w(): number {
    return this.status_x;
  }
  h(): number {
    return HEADER_H;
  }
}
