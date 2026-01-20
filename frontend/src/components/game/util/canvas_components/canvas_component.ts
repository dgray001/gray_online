import type { BoardTransformData } from '../canvas_board/canvas_board';
import type { Point2D } from '../objects2d';

/** Data a canvas component must have */
export declare interface CanvasComponent {
  isHovering: () => boolean;
  isClicking: () => boolean;
  draw: (ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number) => void;
  // should return true if scroll had an effect on the component
  scroll?: (dy: number, dx?: number) => boolean;
  // should return true if the component is hovered
  mousemove: (m: Point2D, transform: BoardTransformData) => boolean;
  // should return true if mousedown had an effect on the component
  mousedown: (e: MouseEvent) => boolean;
  mouseup: (e: MouseEvent) => void;
  xi: () => number;
  xf: () => number;
  yi: () => number;
  yf: () => number;
  w: () => number;
  h: () => number;
}

/** Config for drawing */
export declare interface DrawConfig {
  /** sets canvas fillStyle */
  fill_style: string;
  /** sets canvas strokeStyle */
  stroke_style: string;
  /** sets canvas lineWidth */
  stroke_width: number;
  /** sets canvas fillStyle when hovered */
  hover_fill_style?: string;
  /** sets canvas strokeStyle when hovered */
  hover_stroke_style?: string;
  /** sets canvas lineWidth when hovered */
  hover_stroke_width?: number;
  /** sets canvas fillStyle when clicked */
  click_fill_style?: string;
  /** sets canvas strokeStyle when clicked */
  click_stroke_style?: string;
  /** sets canvas lineWidth when clicked */
  click_stroke_width?: number;
  /** draw with clicked config when clicked and not hovered */
  draw_clicked_when_unhovered?: boolean;
  /** forces draw to ignore transformations */
  fixed_position?: boolean;
}

/** Applies input draw config to the input canvas */
export function configDraw(
  ctx: CanvasRenderingContext2D,
  transform: BoardTransformData,
  config: DrawConfig,
  hovering: boolean,
  clicking: boolean,
  draw: () => void
) {
  if (clicking) {
    if (hovering) {
      ctx.fillStyle = config.click_fill_style ?? config.hover_fill_style ?? config.fill_style;
      ctx.strokeStyle = config.click_stroke_style ?? config.hover_stroke_style ?? config.stroke_style;
      ctx.lineWidth = config.click_stroke_width ?? config.hover_stroke_width ?? config.stroke_width;
    } else if (config.draw_clicked_when_unhovered) {
      ctx.fillStyle = config.click_fill_style ?? config.fill_style;
      ctx.strokeStyle = config.click_stroke_style ?? config.stroke_style;
      ctx.lineWidth = config.click_stroke_width ?? config.stroke_width;
    } else {
      ctx.fillStyle = config.fill_style;
      ctx.strokeStyle = config.stroke_style;
      ctx.lineWidth = config.stroke_width;
    }
  } else if (hovering) {
    ctx.fillStyle = config.hover_fill_style ?? config.fill_style;
    ctx.strokeStyle = config.hover_stroke_style ?? config.stroke_style;
    ctx.lineWidth = config.hover_stroke_width ?? config.stroke_width;
  } else {
    ctx.fillStyle = config.fill_style;
    ctx.strokeStyle = config.stroke_style;
    ctx.lineWidth = config.stroke_width;
  }
  if (config.fixed_position) {
    ctx.scale(1 / transform.scale, 1 / transform.scale);
    ctx.translate(transform.view.x, transform.view.y);
  }
  ctx.beginPath();
  draw();
  if (config.fixed_position) {
    ctx.translate(-transform.view.x, -transform.view.y);
    ctx.scale(transform.scale, transform.scale);
  }
}

/** Data describing a rotation */
export declare interface Rotation {
  /** should be true for counterclockwise */
  direction: boolean;
  /** can represent an angle move or a target angle */
  angle: number;
}
