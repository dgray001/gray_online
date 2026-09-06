import type { ColorRGB } from '../../../../scripts/color_rgb';

/** RGB value reserved in unit/building art to mark pixels swapped for the owning player's color */
export const PLAYER_COLOR_KEY = { r: 254, g: 12, b: 203 };

/** Standard native resolution for unit/building art */
export const PLAYER_ICON_SIZE = 256;

export class RisqImageCache {
  private canvases = new Map<string, HTMLCanvasElement>();
  private cursor_urls = new Map<string, string>();

  private buildImage(
    size: number,
    images: HTMLImageElement[],
    draw: (ctx: CanvasRenderingContext2D) => void
  ): HTMLCanvasElement | undefined {
    if (!images.every((img) => img.complete)) {
      return undefined;
    }
    const canvas = document.createElement('canvas');
    canvas.width = size;
    canvas.height = size;
    const ctx = canvas.getContext('2d');
    if (ctx) {
      draw(ctx);
    }
    return canvas;
  }

  getImage(
    key: string,
    size: number,
    images: HTMLImageElement[],
    draw: (ctx: CanvasRenderingContext2D) => void
  ): HTMLCanvasElement | undefined {
    const cached = this.canvases.get(key);
    if (cached) {
      return cached;
    }
    const canvas = this.buildImage(size, images, draw);
    if (canvas) {
      this.canvases.set(key, canvas);
    }
    return canvas;
  }

  /** Returns img with every PLAYER_COLOR_KEY pixel swapped for color, cached per (key, color) */
  getPlayerColoredIcon(
    key: string,
    img: HTMLImageElement,
    size: number,
    color: ColorRGB
  ): HTMLCanvasElement | undefined {
    return this.getImage(`${key}|${color.getR()},${color.getG()},${color.getB()}`, size, [img], (ctx) => {
      ctx.drawImage(img, 0, 0, size, size);
      const image_data = ctx.getImageData(0, 0, size, size);
      const data = image_data.data;
      for (let i = 0; i < data.length; i += 4) {
        if (
          data[i] === PLAYER_COLOR_KEY.r &&
          data[i + 1] === PLAYER_COLOR_KEY.g &&
          data[i + 2] === PLAYER_COLOR_KEY.b
        ) {
          data[i] = color.getR();
          data[i + 1] = color.getG();
          data[i + 2] = color.getB();
        }
      }
      ctx.putImageData(image_data, 0, 0);
    });
  }

  getCursorUrl(
    key: string,
    size: number,
    images: HTMLImageElement[],
    draw: (ctx: CanvasRenderingContext2D) => void
  ): string | undefined {
    const cached = this.cursor_urls.get(key);
    if (cached) {
      return cached;
    }
    const canvas = this.buildImage(size, images, draw);
    if (!canvas) {
      return undefined;
    }
    const url = canvas.toDataURL();
    this.cursor_urls.set(key, url);
    return url;
  }
}
