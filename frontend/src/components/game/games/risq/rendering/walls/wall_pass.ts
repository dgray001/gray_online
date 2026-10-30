import { isImageReady } from '../../../../../../scripts/image';
import type { Point2D } from '../../../../util/objects2d';
import { rotatePoint } from '../../../../util/objects2d';
import type { RisqBuilding, RisqSpace, RisqZone } from '../../model/types';
import { RisqVisibilityLevel } from '../../model/types';
import type { RisqDrawHost } from '../board/board_renderer';
import { OUTER_ZONE_INDICES } from '../zones/geometry';
import type { RisqViewMode } from '../terrain';
import { buildingVisibleInViewMode } from '../terrain';
import type { WallSegmentInput } from './wall_geometry';
import { wallMetrics } from './wall_metrics';
import type { WallDrawItem, WallTextures } from './wall_draw';
import { wallDrawItems } from './wall_draw';
import { WallSpriteCache } from './wall_sprites';
import { isCompletedWall, isWallRubble, wallAlpha } from './wall_state';
import type { WallDamage } from './wall_damage';
import { transformScale } from '../assets/image_cache';
import { damageLevel } from '../assets/buildings';

function segmentDamage(zone: RisqZone | undefined, building: RisqBuilding | undefined): WallDamage {
  if (building && isCompletedWall(building)) {
    return damageLevel(building.combat_stats);
  }
  if (!zone || !isWallRubble(zone)) {
    return 0;
  }
  return (zone.destroyed_building_turns ?? 1) >= 2 ? 4 : 3;
}

const SPRITE_BUDGET_BYTES = 48 * 1024 * 1024;
const ROTATION_SETTLE_MS = 150;
const DEFAULT_COLOR = 'rgb(44, 88, 190)';
const JOIN_CELL_RATIO = 0.1;
const JOIN_DISTANCE_RATIO = 0.02;
const FRONT_FACING = 0.02;

export declare interface WallPassConfig {
  hex_r: number;
  rotation: number;
  view_mode: RisqViewMode;
}

export type WallPhase = 'back' | 'front';

declare interface CanvasSegment {
  space: RisqSpace;
  a: Point2D;
  b: Point2D;
  normal: Point2D;
  color: string;
  alpha: number;
  damage: WallDamage;
  face: number;
  garrison_icon?: CanvasImageSource;
}

declare interface PhaseItems {
  back: WallDrawItem[];
  front: WallDrawItem[];
}

function canvasSegments(spaces: RisqSpace[], host: RisqDrawHost, config: WallPassConfig): CanvasSegment[] {
  const segments: CanvasSegment[] = [];
  for (const space of spaces) {
    if (!space.zones || space.visibility < RisqVisibilityLevel.FOG) {
      continue;
    }
    for (const [i, index] of OUTER_ZONE_INDICES.entries()) {
      const zone = space.zones[index.x][index.y];
      const alpha = zone ? wallAlpha(host, zone) : undefined;
      const building = zone?.building;
      if (alpha === undefined || (building && !buildingVisibleInViewMode(building, config.view_mode))) {
        continue;
      }
      const vertex = (j: number): Point2D => {
        const angle = Math.PI / 6 + (Math.PI / 3) * j;
        return {
          x: space.center.x + config.hex_r * Math.cos(angle),
          y: space.center.y + config.hex_r * Math.sin(angle),
        };
      };
      const normal_angle = (Math.PI / 3) * (i + 1);
      const color = host.getGame()?.players[building?.player_id ?? host.getPlayerId()]?.color;
      segments.push({
        space,
        alpha,
        damage: segmentDamage(zone, building),
        face: i,
        a: vertex(i),
        b: vertex(i + 1),
        normal: { x: Math.cos(normal_angle), y: Math.sin(normal_angle) },
        color: color?.getString() ?? DEFAULT_COLOR,
        garrison_icon:
          color && isCompletedWall(building) && building?.has_garrisoned_units
            ? host.getPlayerColoredIcon('risq/icons/garrison_flag', color)
            : undefined,
      });
    }
  }
  return segments;
}

class VertexIndex {
  private cells = new Map<string, { segment: number; point: Point2D }[]>();

  constructor(
    segments: CanvasSegment[],
    private cell: number,
    private reach: number
  ) {
    for (const [segment, s] of segments.entries()) {
      for (const point of [s.a, s.b]) {
        const key = this.key(point);
        const bucket = this.cells.get(key) ?? [];
        bucket.push({ segment, point });
        this.cells.set(key, bucket);
      }
    }
  }

  private key(p: Point2D): string {
    return `${Math.round(p.x / this.cell)},${Math.round(p.y / this.cell)}`;
  }

  neighborOf(segment: number, p: Point2D): number | undefined {
    const cx = Math.round(p.x / this.cell);
    const cy = Math.round(p.y / this.cell);
    for (let dx = -1; dx <= 1; dx++) {
      for (let dy = -1; dy <= 1; dy++) {
        for (const entry of this.cells.get(`${cx + dx},${cy + dy}`) ?? []) {
          if (entry.segment !== segment && Math.hypot(entry.point.x - p.x, entry.point.y - p.y) < this.reach) {
            return entry.segment;
          }
        }
      }
    }
    return undefined;
  }
}

function uprightSegments(segments: CanvasSegment[], config: WallPassConfig): WallSegmentInput[] {
  const index = new VertexIndex(segments, JOIN_CELL_RATIO * config.hex_r, JOIN_DISTANCE_RATIO * config.hex_r);
  const upright = segments.map((s) => rotatePoint(s.normal, config.rotation));
  return segments.map((s, i) => {
    const start = index.neighborOf(i, s.a);
    const end = index.neighborOf(i, s.b);
    return {
      a: rotatePoint(s.a, config.rotation),
      b: rotatePoint(s.b, config.rotation),
      normal: upright[i],
      color: s.color,
      alpha: s.alpha,
      damage: s.damage,
      face: s.face,
      garrison_icon: s.garrison_icon,
      start_neighbor_normal: start === undefined ? undefined : upright[start],
      end_neighbor_normal: end === undefined ? undefined : upright[end],
    };
  });
}

export class RisqWallPass {
  private cache = new WallSpriteCache(SPRITE_BUDGET_BYTES);
  private last_rotation = Number.NaN;
  private last_hex_r = Number.NaN;
  private live_until = 0;
  private live = false;
  private rotation = 0;
  private prepared = new Map<RisqSpace, PhaseItems>();

  reset(): void {
    this.prepared.clear();
  }

  prepare(host: RisqDrawHost, spaces: RisqSpace[], config: WallPassConfig): void {
    this.prepared.clear();
    const textures = this.textures(host);
    if (!textures) {
      return;
    }
    this.beginFrame(config);
    const segments = canvasSegments(spaces, host, config);
    if (segments.length === 0) {
      return;
    }
    const metrics = wallMetrics(config.hex_r);
    for (const [i, input] of uprightSegments(segments, config).entries()) {
      const space = segments[i].space;
      const phases = this.prepared.get(space) ?? { back: [], front: [] };
      const phase = input.normal.y > FRONT_FACING ? phases.front : phases.back;
      phase.push(...wallDrawItems(textures, metrics, input));
      this.prepared.set(space, phases);
    }
    for (const phases of this.prepared.values()) {
      phases.back.sort((p, q) => p.y - q.y);
      phases.front.sort((p, q) => p.y - q.y);
    }
  }

  draw(ctx: CanvasRenderingContext2D, space: RisqSpace, phase: WallPhase): void {
    const items = this.prepared.get(space)?.[phase];
    if (!items || items.length === 0) {
      return;
    }
    ctx.save();
    ctx.rotate(-this.rotation);
    const scale = transformScale(ctx);
    for (const item of items) {
      ctx.globalAlpha = item.alpha;
      if (this.live || !this.cache.draw(ctx, item, scale)) {
        ctx.save();
        ctx.translate(item.anchor.x, item.anchor.y);
        item.draw(ctx);
        ctx.restore();
      }
    }
    ctx.restore();
  }

  private beginFrame(config: WallPassConfig): void {
    const now = performance.now();
    this.rotation = config.rotation;
    if (config.rotation !== this.last_rotation || config.hex_r !== this.last_hex_r) {
      this.cache.clear();
      this.live_until = now + ROTATION_SETTLE_MS;
      this.last_rotation = config.rotation;
      this.last_hex_r = config.hex_r;
    }
    this.live = now < this.live_until;
  }

  private textures(host: RisqDrawHost): WallTextures | undefined {
    const load = (name: string): HTMLImageElement => host.getIcon(`risq/walls/${name}`);
    const stone = load('stone');
    const deck = load('deck');
    const tower = {
      stone: load('tower_stone'),
      frame_front: load('tower_frame_front'),
      frame_side: load('tower_frame_side'),
      pal: load('tower_pal'),
      top: load('tower_top'),
      thatch: load('thatch'),
    };
    const rubble = load('rubble_stones');
    const ready = [stone, deck, rubble, ...Object.values(tower)].every(isImageReady);
    return ready ? { stone, deck, rubble, tower } : undefined;
  }
}
