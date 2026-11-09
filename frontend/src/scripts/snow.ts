interface Snowflake {
  x: number;
  y: number;
  radius: number;
  speed_y: number;
  speed_x: number;
  wobble: number;
}

export class SnowOverlay {
  private canvas: HTMLCanvasElement;
  private ctx: CanvasRenderingContext2D;
  private flakes: Snowflake[] = [];
  private anim_id = 0;

  constructor() {
    this.canvas = document.createElement('canvas');
    this.canvas.id = 'snow-canvas';
    this.canvas.style.cssText = 'position:fixed;inset:0;pointer-events:none;z-index:90;';
    document.body.appendChild(this.canvas);
    this.ctx = this.canvas.getContext('2d') as CanvasRenderingContext2D;
    this.resize();
    window.addEventListener('resize', (): void => this.resize());
    this.initFlakes();
    this.tick();
  }

  private resize(): void {
    this.canvas.width = window.innerWidth;
    this.canvas.height = window.innerHeight;
  }

  private initFlakes(): void {
    const count = Math.min(80, Math.floor(window.innerWidth / 20));
    this.flakes = [];
    for (let i = 0; i < count; i++) {
      this.flakes.push({
        x: Math.random() * window.innerWidth,
        y: Math.random() * window.innerHeight,
        radius: Math.random() * 2.5 + 1.2,
        speed_y: Math.random() * 1.2 + 0.6,
        speed_x: (Math.random() - 0.5) * 0.5,
        wobble: Math.random() * Math.PI * 2,
      });
    }
  }

  private tick(): void {
    this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
    this.ctx.fillStyle = 'rgba(255, 255, 255, 0.85)';
    for (const flake of this.flakes) {
      flake.wobble += 0.02;
      flake.x += flake.speed_x + Math.sin(flake.wobble) * 0.5;
      flake.y += flake.speed_y;
      if (flake.y > this.canvas.height) {
        flake.y = -5;
        flake.x = Math.random() * this.canvas.width;
      }
      this.ctx.beginPath();
      this.ctx.arc(flake.x, flake.y, flake.radius, 0, Math.PI * 2);
      this.ctx.fill();
    }
    this.anim_id = requestAnimationFrame((): void => this.tick());
  }
}

let snow_instance: SnowOverlay | undefined;

export function startSnow(): void {
  if (!snow_instance) {
    snow_instance = new SnowOverlay();
  }
}
