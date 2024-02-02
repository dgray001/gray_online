import {BoardTransformData} from '../../canvas_board/canvas_board';
import {Point2D, addPoint2D, equalsPoint2D} from '../../objects2d';
import {DrawConfig, Rotation, configDraw} from '../canvas_component';
import {ButtonConfig, DwgButton} from './button';

/** Config data for a rect button */
export declare interface RectButtonConfig {
  button_config: ButtonConfig;
  p: Point2D;
  w: number;
  h: number;
  draw_config: DrawConfig;
  move_animation_speed?: number;
  rotate_animation_speed?: number;
  image_path?: string;
  rotation?: number;
}

export abstract class DwgRectButton extends DwgButton {
  private rect_config: RectButtonConfig;
  private center_p: Point2D;
  private radius_p: Point2D;
  private speed_p: Point2D;
  private target_p: Point2D;
  private target_callback: () => void;
  private reached_target = {x: false, y: false};
  private rotate_target: Rotation;
  private rotate_speed: number;
  private rotate_reached: boolean;
  private rotate_callback: () => void;
  private img: HTMLImageElement;

  constructor(config: RectButtonConfig) {
    super(config.button_config);
    config.move_animation_speed = config.move_animation_speed ?? 0;
    config.rotate_animation_speed = config.rotate_animation_speed ?? 0;
    if (!!config.image_path) {
      this.img = document.createElement('img');
      this.img.src = `/images/${config.image_path}.png`;
      this.img.draggable = false;
    }
    config.rotation = config.rotation ?? 0;
    this.rect_config = config;
    this.refreshPositionDependencies();
  }

  setPosition(p: Point2D, callback?: () => void, no_animation = false) {
    if (this.rect_config.move_animation_speed > 0 && !equalsPoint2D(p, this.rect_config.p) && !no_animation) {
      this.target_p = p;
      this.speed_p = {
        x: (p.x - this.rect_config.p.x) / this.rect_config.move_animation_speed,
        y: (p.y - this.rect_config.p.y) / this.rect_config.move_animation_speed,
      };
      this.reached_target = {x: false, y: false};
      this.target_callback = callback;
    } else {
      this.rect_config.p = p;
      this.refreshPositionDependencies();
      if (!!callback) {
        callback();
      }
    }
  }

  setRotation(rotation: Rotation, callback?: () => void, no_animation = false) {
    if (this.rect_config.rotate_animation_speed > 0 && rotation.angle !== this.rect_config.rotation && !no_animation) {
      this.rotate_target = rotation;
      const rotation_needed = (rotation.angle - this.rect_config.rotation) % (2 * Math.PI);
      this.rotate_speed = rotation_needed / this.rect_config.rotate_animation_speed;
      this.rotate_reached = false;
      this.rotate_callback = callback;
    } else {
      this.rect_config.rotation = rotation.angle;
      if (!!callback) {
        callback();
      }
    }
  }

  private refreshPositionDependencies() {
    this.radius_p = {
      x: 0.5 * this.rect_config.w,
      y: 0.5 * this.rect_config.h,
    };
    this.center_p = addPoint2D(this.rect_config.p, this.radius_p);
  }

  xi(): number {
    return this.rect_config.p.x;
  }
  yi(): number {
    return this.rect_config.p.y;
  }
  xf(): number {
    return this.rect_config.p.x + this.rect_config.w;
  }
  yf(): number {
    return this.rect_config.p.y + this.rect_config.h;
  }
  xc(): number {
    return this.center_p.x;
  }
  yc(): number {
    return this.center_p.y;
  }
  w(): number {
    return this.rect_config.w;
  }
  h(): number {
    return this.rect_config.h;
  }

  protected override _draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    // translate animation
    if (!!this.target_p && !!this.speed_p) {
      if (this.reached_target.x && this.reached_target.y) {
        this.rect_config.p = {...this.target_p};
        this.target_p = undefined;
        this.speed_p = undefined;
        if (!!this.target_callback) {
          this.target_callback();
        }
      } else {
        if (!this.reached_target.x) {
          const dx = this.speed_p.x * dt;
          if (Math.abs(this.rect_config.p.x - this.target_p.x) <= Math.abs(dx)) {
            this.reached_target.x = true;
            this.rect_config.p.x = this.target_p.x;
          } else {
            this.rect_config.p.x += dx;
            if (Math.abs(this.rect_config.p.x - this.target_p.x) <= Math.abs(dx)) {
              this.reached_target.x = true;
            }
          }
        }
        if (!this.reached_target.y) {
          const dy = this.speed_p.y * dt;
          if (Math.abs(this.rect_config.p.y - this.target_p.y) <= Math.abs(dy)) {
            this.reached_target.y = true;
            this.rect_config.p.y = this.target_p.y;
          } else {
            this.rect_config.p.y += dy;
            if (Math.abs(this.rect_config.p.y - this.target_p.y) <= Math.abs(dy)) {
              this.reached_target.y = true;
            }
          }
        }
      }
      this.refreshPositionDependencies();
    }
    // rotate animation
    if (!!this.rotate_target && !!this.rotate_speed) {
      const da = this.rotate_speed * dt;
      if (this.rotate_reached || Math.abs(this.rect_config.rotation - this.rotate_target.angle) <= Math.abs(da)) {
        this.rect_config.rotation = this.rotate_target.angle;
        this.rotate_target = undefined;
        this.rotate_speed = undefined;
      } else {
        this.rect_config.rotation += da;
        if (Math.abs(this.rect_config.rotation - this.rotate_target.angle) <= Math.abs(da)) {
          this.rotate_reached = true;
        }
      }
    }
    // actual draw
    configDraw(ctx, transform, this.rect_config.draw_config, this.isHovering(), this.isClicking(), () => {
      ctx.translate(this.center_p.x, this.center_p.y);
      ctx.rotate(this.rect_config.rotation);
      if (!!this.img) {
        ctx.drawImage(this.img, -this.radius_p.x, -this.radius_p.y, this.rect_config.w, this.rect_config.h);
      }
      ctx.rect(-this.radius_p.x, -this.radius_p.y, this.rect_config.w, this.rect_config.h);
      ctx.rotate(-this.rect_config.rotation);
      ctx.translate(-this.center_p.x, -this.center_p.y);
    });
  }

  override mouseOver(m: Point2D, transform: BoardTransformData): boolean {
    if (this.rect_config.draw_config.fixed_position) {
      m = {
        x: m.x * transform.scale - transform.view.x,
        y: m.y * transform.scale - transform.view.y,
      };
    }
    if (m.x < this.xi() || m.y < this.yi() || m.x > this.xf() || m.y > this.yf()) {
      return false;
    }
    return true;
  }
}
