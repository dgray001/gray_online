import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { DwgRectButton } from '../../../../util/canvas_components/button/rect_button';
import { drawText } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import type { ReplayEvent } from '../../model/replay_event';
import type { DwgRisq } from '../../risq';
import { eventPoint, overlaps, panelPosition } from './event_geometry';
import { RisqEventButton } from './event_button';
import { drawBattleDetails } from './battle_details';
import { rangePaths } from '../../rendering/board/attack_ranges';
import { zoneVertices } from '../../rendering/zones/geometry';
import type { RisqTickLocation } from '../../model/tick_actions';
import { ColorRGB } from '../../../../../../scripts/color_rgb';

type EventMarker = { event: ReplayEvent; button: RisqEventButton };
const ELIMINATION_COLOR = new ColorRGB(210, 35, 35);

export class RisqReplayEvents extends DwgRectButton {
  private source?: readonly ReplayEvent[];
  private markers: EventMarker[] = [];
  private selected: EventMarker[] = [];
  private selected_index = 0;
  private panel_visible = false;
  private zone_highlight?: { event: ReplayEvent; radius: number; fill: Path2D; outline: Path2D };
  private jump = new RisqEventButton(undefined, 'Jump to tick', (): void => this.jumpToSelected(), 120);
  private next = new RisqEventButton(
    undefined,
    'Next',
    (): void => {
      this.selected_index = (this.selected_index + 1) % this.selected.length;
      this.jumpToSelected();
    },
    60
  );
  private close = new RisqEventButton(
    undefined,
    'Close',
    (): void => {
      this.selected = [];
      this.panel_visible = false;
    },
    60
  );

  constructor(
    private risq: DwgRisq,
    private hud: () => CanvasComponent[]
  ) {
    super({
      button_config: {},
      p: { x: 0, y: 0 },
      w: 280,
      h: 174,
      r: 6,
      draw_config: {
        fill_style: 'rgb(222, 184, 135)',
        stroke_style: 'rgb(70, 30, 5)',
        stroke_width: 2,
        fixed_position: true,
      },
    });
  }

  protected hovered(): void {}
  protected unhovered(): void {}
  protected clicked(): void {}
  protected released(): void {}

  drawZones(ctx: CanvasRenderingContext2D): void {
    const event = this.selected[this.selected_index]?.event;
    if (!this.risq.session.isReplaying() || !event) {
      this.zone_highlight = undefined;
      return;
    }
    const radius = this.risq.viewport.hexR();
    if (this.zone_highlight?.event !== event || this.zone_highlight.radius !== radius) {
      const polygons = event.locations.map((location: RisqTickLocation): Point2D[] => {
        const center = this.risq.viewport.coordinateToCanvas(location.space);
        return zoneVertices(location.zone, radius).map(
          (point: Point2D): Point2D => ({ x: center.x + point.x, y: center.y + point.y })
        );
      });
      this.zone_highlight = { event, radius, ...rangePaths(polygons) };
    }
    ctx.save();
    ctx.setLineDash([]);
    ctx.fillStyle = 'rgba(255, 255, 255, 0.12)';
    ctx.fill(this.zone_highlight.fill);
    ctx.strokeStyle = 'white';
    ctx.lineWidth = 2 / this.risq.viewport.lastTransform().scale;
    ctx.stroke(this.zone_highlight.outline);
    ctx.restore();
  }

  reset(): void {
    this.zone_highlight = undefined;
    this.setClicking(false);
    this.setHovering(false);
    this.source = undefined;
    this.selected_index = 0;
    this.markers = [];
    this.selected = [];
    this.panel_visible = false;
  }

  private layout(transform: BoardTransformData): void {
    const events = this.risq.session.getReplayEvents();
    if (events !== this.source) {
      this.reset();
      this.source = events;
      this.markers = events.map((event: ReplayEvent): EventMarker => {
        const marker: EventMarker = {
          event,
          button: new RisqEventButton(this.risq.getIcon(event.icon), '', (): void => this.select(marker)),
        };
        marker.button.setSize(64, 64);
        return marker;
      });
    }
    for (const { event, button } of this.markers) {
      if (event.kind === 'eliminated') {
        const icon = this.risq.getIcon(event.icon);
        const tinted = this.risq.getImageCache().getTintedIcon(event.icon, icon, 64, ELIMINATION_COLOR);
        if (tinted) {
          button.setImage(tinted);
        }
      }
      const point = eventPoint(event, this.risq.viewport, transform, this.risq.hudScale(), this.risq.hudCanvasSize());
      button.setPosition({ x: point.x - button.w() / 2, y: point.y - button.h() / 2 });
    }
    const selected = this.selected[this.selected_index];
    this.setSize(
      selected?.event.kind === 'battle' || selected?.event.kind === 'disruption' ? 320 : 280,
      selected?.event.kind === 'battle' ? 232 + 20 * Math.max(1, (selected.event.battle_stats?.length ?? 1) - 1) : 174
    );
    const point =
      selected &&
      eventPoint(selected.event, this.risq.viewport, transform, this.risq.hudScale(), this.risq.hudCanvasSize());
    const position = point && panelPosition(point, { x: this.w(), y: this.h() }, this.risq.hudCanvasSize(), this.hud());
    this.panel_visible = !!position;
    if (position) {
      this.setPosition(position);
      [this.jump, this.next, this.close].forEach((button: RisqEventButton, i: number): void =>
        button.setPosition({ x: position.x + [8, 136, 204][i], y: position.y + this.h() - 40 })
      );
    }
  }

  private select(marker: EventMarker): void {
    const rect = (button: RisqEventButton): { x: number; y: number; w: number; h: number } => ({
      x: button.xi(),
      y: button.yi(),
      w: button.w(),
      h: button.h(),
    });
    this.selected = this.markers.filter((other: EventMarker): boolean =>
      overlaps(rect(marker.button), rect(other.button))
    );
    this.selected_index = this.selected.indexOf(marker);
    this.jumpToSelected();
  }

  private jumpToSelected(): void {
    const event = this.selected[this.selected_index].event;
    this.risq.replay_panel.seek(event.kind === 'battle' ? event.tick - 0.5 : event.tick);
  }

  protected override _draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    if (!this.risq.session.isReplaying()) {
      return;
    }
    this.layout(transform);
    for (const marker of this.markers) {
      marker.button.draw(ctx, transform, dt);
    }
    if (!this.panel_visible) {
      return;
    }
    super._draw(ctx, transform, dt);
    this.drawDetails(ctx, transform);
    for (const button of [this.jump, this.next, this.close]) {
      button.draw(ctx, transform, dt);
    }
  }

  private drawDetails(ctx: CanvasRenderingContext2D, transform: BoardTransformData): void {
    const event = this.selected[this.selected_index].event;
    const nickname = this.risq.getGame()?.players[event.player_id]?.player.nickname ?? `Player ${event.player_id}`;
    const lines =
      event.kind === 'eliminated'
        ? [nickname, 'Eliminated', 'End of turn']
        : [
            event.kind === 'disruption' ? event.title : `Production ${this.selected_index + 1}/${this.selected.length}`,
            event.kind === 'disruption' ? (event.reason ?? '') : event.title,
            nickname,
            event.end_of_turn ? 'End of turn' : `Tick ${event.tick}`,
          ];
    configDraw(
      ctx,
      transform,
      { fill_style: 'black', stroke_width: 0, fixed_position: true },
      false,
      false,
      (): void => {
        if (event.kind === 'battle') {
          drawBattleDetails(ctx, this.risq, event, this.xi() + 8, this.yi() + 8, this.w() - 16);
          return;
        }
        lines.forEach((line: string, i: number): void =>
          drawText(ctx, line, {
            p: { x: this.xi() + 8, y: this.yi() + 8 + 24 * i },
            w: this.w() - 16,
            font: '18px sans-serif',
            fill_style:
              event.kind === 'eliminated' && i === 0
                ? (this.risq.getGame()?.players[event.player_id]?.color.getString() ?? 'black')
                : 'black',
          })
        );
      }
    );
  }

  scroll(): boolean {
    return this.risq.session.isReplaying() && this.panel_visible && super.isHovering();
  }

  override isHovering(): boolean {
    return (
      this.risq.session.isReplaying() &&
      (super.isHovering() || this.markers.some((marker: EventMarker): boolean => marker.button.isHovering()))
    );
  }
  override setHovering(hovering: boolean): void {
    super.setHovering(hovering);
    if (!hovering) {
      for (const button of [
        ...this.markers.map((marker: EventMarker): RisqEventButton => marker.button),
        this.jump,
        this.next,
        this.close,
      ]) {
        button.setHovering(false);
      }
    }
  }

  override mouseOver(canvas: Point2D, screen: Point2D): boolean {
    return this.panel_visible && super.mouseOver(canvas, screen);
  }
  override mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    if (!this.risq.session.isReplaying()) {
      this.setHovering(false);
      return false;
    }
    this.layout(transform);
    const panel_hit = super.mousemove(canvas, screen, transform);
    for (const button of [this.jump, this.next, this.close]) {
      if (this.panel_visible) {
        button.mousemove(canvas, screen, transform);
      } else {
        button.setHovering(false);
      }
    }
    let claimed = panel_hit;
    for (const { button } of [...this.markers].reverse()) {
      if (claimed) {
        button.setHovering(false);
      } else {
        claimed = button.mousemove(canvas, screen, transform);
      }
    }
    return claimed;
  }

  override mousedown(e: MouseEvent): boolean {
    if (!this.risq.session.isReplaying()) {
      return false;
    }
    if (super.isHovering()) {
      [this.jump, this.next, this.close].some((button: RisqEventButton): boolean => button.mousedown(e));
      return true;
    }
    const marker = this.markers.find((marker: EventMarker): boolean => marker.button.isHovering());
    if (marker) {
      marker.button.mousedown(e);
      return true;
    }
    if (this.selected.length > 0 && e.button === 0) {
      this.selected = [];
      this.panel_visible = false;
      return true;
    }
    return false;
  }
  override mouseup(e: MouseEvent): void {
    for (const button of [
      this.jump,
      this.next,
      this.close,
      ...this.markers.map((marker: EventMarker): RisqEventButton => marker.button),
    ]) {
      button.mouseup(e);
    }
  }
  override setClicking(clicking: boolean): void {
    super.setClicking(clicking);
    if (!clicking) {
      for (const button of [
        this.jump,
        this.next,
        this.close,
        ...this.markers.map((marker: EventMarker): RisqEventButton => marker.button),
      ]) {
        button.setClicking(false);
      }
    }
  }

  override isClicking(): boolean {
    return [
      this.jump,
      this.next,
      this.close,
      ...this.markers.map((marker: EventMarker): RisqEventButton => marker.button),
    ].some((button: RisqEventButton): boolean => button.isClicking());
  }
}
