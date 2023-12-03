
declare interface ColorRGBData {
  r: number;
  g: number;
  b: number;
  a: number;
}

/** Class describing an RGB color */
export class ColorRGB {
  private data: ColorRGBData = {r: 0, g: 0, b: 0, a: 0};

  constructor(r: number, g: number, b:number, a?: number) {
    this.setColor(r, g, b, a);
  }

  setColor(r: number, g: number, b:number, a?: number) {
    this.data = this.cleanInput(r, g, b, a);
  }

  getString(): string {
    return `rgb(${this.data.r}, ${this.data.g}, ${this.data.b}, ${this.data.a})`;
  }

  addColor(r: number, g: number, b:number, a?: number): ColorRGB {
    const input = this.cleanInput(r, g, b, a);
    const new_a = this.data.a + input.a;
    if (new_a === 0) {
      this.data.a = 0;
      return;
    }
    const new_r = (this.data.a * this.data.r + input.a * input.r) / new_a;
    const new_g = (this.data.a * this.data.g + input.a * input.g) / new_a;
    const new_b = (this.data.a * this.data.b + input.a * input.b) / new_a;
    this.data = this.cleanInput(new_r, new_g, new_b, new_a);
    return this;
  }

  dAlpha(da: number): ColorRGB {
    this.data.a += da;
    this.cleanData();
    return this;
  }

  private cleanInput(r: number, g: number, b:number, a?: number): ColorRGBData {
    if (r < 0) {
      r = 0;
    } else if (r > 255) {
      r = 255;
    }
    if (g < 0) {
      g = 0;
    } else if (g > 255) {
      g = 255;
    }
    if (b < 0) {
      b = 0;
    } else if (b > 255) {
      b = 255;
    }
    if (a === undefined) {
      a = 1;
    } else if (a < 0) {
      a = 0;
    } else if (a > 1) {
      a = 1;
    }
    return {r, g, b, a};
  }

  private cleanData() {
    this.data = this.cleanInput(this.data.r, this.data.g, this.data.b, this.data.a);
  }
}
