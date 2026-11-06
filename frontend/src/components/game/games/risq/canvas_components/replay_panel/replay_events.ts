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
import { hexagonVertices, zoneVertices } from '../../rendering/zones/geometry';
import type { RisqTickLocation } from '../../model/tick_actions';
import { ColorRGB } from '../../../../../../scripts/color_rgb';
import { RisqViewMode } from '../../rendering/terrain';
import { RisqReplayCamera } from './replay_camera';
import { drawHiredDetails } from './hired_details';
import { adjacentReplayEvent, replayEventStop } from '../../application/replay/event_navigation';

type EventMarker = { event: ReplayEvent; button: RisqEventButton };
const ELIMINATION_COLOR = new ColorRGB(210, 35, 35);
const GAIN_COLOR = new ColorRGB(20, 140, 40);

export class RisqReplayEvents extends DwgRectButton {
  private source?: readonly ReplayEvent[];
  private markers: EventMarker[] = [];
  private selected: EventMarker[] = [];
  private selected_index = 0;
  private panel_visible = false;
  private panel_position?: Point2D;
  private drag_offset?: Point2D;
  private pointer_position: Point2D = { x: 0, y: 0 };
  private camera = new RisqReplayCamera((): DwgRisq['board'] => this.risq.board);
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

  private adjacentEvent(direction: -1 | 1): ReplayEvent | undefined {
    return adjacentReplayEvent(
      this.risq.session.getReplayEvents(),
      this.selected[this.selected_index]?.event,
      this.risq.replay_panel.tick(),
      direction
    );
  }

  canStepEvent(direction: -1 | 1): boolean {
    return !!this.adjacentEvent(direction);
  }

  stepEvent(direction: -1 | 1): void {
    const event = this.adjacentEvent(direction);
    if (event) {
      this.open(event);
    }
  }

  open(event: ReplayEvent): boolean {
    if (!this.risq.session.getReplayEvents().includes(event)) {
      return false;
    }
    if (!this.risq.session.isReplaying()) {
      this.risq.toggleReplay();
    }
    this.layout(this.risq.viewport.lastTransform());
    const marker = this.markers.find((item: EventMarker): boolean => item.event === event);
    if (!marker) {
      return false;
    }
    this.selected = [marker];
    this.selected_index = 0;
    this.layout(this.risq.viewport.lastTransform());
    this.select(marker, true);
    this.camera.focus(event, this.risq.viewport);
    return true;
  }

  private isVisible(event: ReplayEvent): boolean {
    if (this.selected[this.selected_index]?.event === event) {
      return true;
    }
    const mode = this.risq.viewport.viewMode();
    if (mode === RisqViewMode.RESOURCE) {
      return event.kind !== 'battle' && event.kind !== 'eliminated';
    }
    if (mode === RisqViewMode.MILITARY) {
      return event.kind !== 'production' || event.production?.source === 'hired';
    }
    if (mode === RisqViewMode.OWNERSHIP) {
      return event.kind !== 'production';
    }
    return true;
  }

  private visibleMarkers(): EventMarker[] {
    const selected = this.selected[this.selected_index];
    return this.markers
      .filter((marker: EventMarker): boolean => this.isVisible(marker.event))
      .sort((a: EventMarker, b: EventMarker): number => Number(a === selected) - Number(b === selected));
  }

  drawZones(ctx: CanvasRenderingContext2D): void {
    const event = this.selected[this.selected_index]?.event;
    if (!this.risq.session.isReplaying() || !event || !this.isVisible(event)) {
      this.zone_highlight = undefined;
      return;
    }
    const radius = this.risq.viewport.hexR();
    if (this.zone_highlight?.event !== event || this.zone_highlight.radius !== radius) {
      const polygons = event.locations.map((location: RisqTickLocation): Point2D[] => {
        const center = this.risq.viewport.coordinateToCanvas(location.space);
        const vertices = event.kind === 'continent' ? hexagonVertices(radius) : zoneVertices(location.zone, radius);
        return vertices.map((point: Point2D): Point2D => ({ x: center.x + point.x, y: center.y + point.y }));
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
    this.camera.cancel();
    this.panel_position = undefined;
    this.drag_offset = undefined;
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
    const active = this.selected[this.selected_index];
    this.selected = this.selected.filter((marker: EventMarker): boolean => this.isVisible(marker.event));
    this.selected_index = active ? Math.max(0, this.selected.indexOf(active)) : 0;
    for (const marker of this.markers) {
      if (!this.isVisible(marker.event)) {
        marker.button.setHovering(false);
        marker.button.setClicking(false);
        marker.button.setSelected(false);
      }
    }
    for (const { event, button } of this.visibleMarkers()) {
      if (event.kind === 'eliminated' || event.kind === 'continent') {
        const icon = this.risq.getIcon(event.icon);
        const color =
          event.kind === 'eliminated' || event.continent?.change === 'lost' ? ELIMINATION_COLOR : GAIN_COLOR;
        const tinted = this.risq.getImageCache().getTintedIcon(event.icon, icon, 64, color);
        if (tinted) {
          button.setImage(tinted);
        }
      }
      const point = eventPoint(event, this.risq.viewport, transform, this.risq.hudScale(), this.risq.hudCanvasSize());
      button.setPosition({ x: point.x - button.w() / 2, y: point.y - button.h() / 2 });
    }
    const selected = this.selected[this.selected_index];
    this.setSize(
      selected?.event.kind === 'battle' || selected?.event.kind === 'disruption' || selected?.event.hired_units
        ? 320
        : 280,
      selected?.event.kind === 'battle'
        ? 260 + 20 * Math.max(1, (selected.event.battle_stats?.length ?? 1) - 1)
        : selected?.event.hired_units
          ? 140 + selected.event.hired_units.length * 32 + (selected.event.location_name ? 24 : 0)
          : 174
    );
    const point =
      selected &&
      eventPoint(selected.event, this.risq.viewport, transform, this.risq.hudScale(), this.risq.hudCanvasSize());
    const size = this.risq.hudCanvasSize();
    const position =
      selected && this.panel_position
        ? {
            x: Math.max(0, Math.min(this.panel_position.x, size.width - this.w())),
            y: Math.max(0, Math.min(this.panel_position.y, size.height - this.h())),
          }
        : point && panelPosition(point, { x: this.w(), y: this.h() }, size, this.hud());
    this.panel_visible = !!position;
    if (position) {
      this.setPosition(position);
      [this.jump, this.next, this.close].forEach((button: RisqEventButton, i: number): void =>
        button.setPosition({ x: position.x + [8, 136, 204][i], y: position.y + this.h() - 40 })
      );
    }
  }

  private select(marker: EventMarker, opening: boolean = false): void {
    if (!opening && this.selected[this.selected_index] === marker) {
      this.selected = [];
      this.panel_visible = false;
      marker.button.setSelected(false);
      return;
    }
    this.panel_position = undefined;
    const rect = (button: RisqEventButton): { x: number; y: number; w: number; h: number } => ({
      x: button.xi(),
      y: button.yi(),
      w: button.w(),
      h: button.h(),
    });
    this.selected = this.visibleMarkers().filter((other: EventMarker): boolean =>
      overlaps(rect(marker.button), rect(other.button))
    );
    this.selected_index = this.selected.indexOf(marker);
    this.jumpToSelected();
  }

  private jumpToSelected(): void {
    const event = this.selected[this.selected_index].event;
    this.risq.replay_panel.seek(replayEventStop(event));
  }

  protected override _draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    if (!this.risq.session.isReplaying()) {
      return;
    }
    this.layout(transform);
    for (const marker of this.visibleMarkers()) {
      marker.button.setSelected(marker === this.selected[this.selected_index]);
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
        : event.continent
          ? [
              event.title,
              nickname,
              `${event.continent.change === 'gained' ? '+' : '-'}${Math.round(event.continent.gold_bonus)} gold/turn`,
              'End of turn',
            ]
          : [
              event.kind === 'disruption'
                ? event.title
                : `Production ${this.selected_index + 1}/${this.selected.length}`,
              event.kind === 'disruption' ? (event.reason ?? '') : event.title,
              nickname,
              event.end_of_turn ? 'End of turn' : `Tick ${event.tick}`,
            ];
    if (event.location_name) {
      lines.splice(event.kind === 'eliminated' ? 2 : 3, 0, event.location_name);
    }
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
        if (event.hired_units) {
          drawHiredDetails(ctx, this.risq, event, this.xi() + 8, this.yi() + 8, this.w() - 16);
          return;
        }
        lines.forEach((line: string, i: number): void =>
          drawText(ctx, line, {
            p: { x: this.xi() + 8, y: this.yi() + 8 + 24 * i },
            w: this.w() - 16,
            font: '18px sans-serif',
            fill_style:
              (event.kind === 'eliminated' && i === 0) || (event.kind === 'continent' && i === 1)
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
      (super.isHovering() || this.visibleMarkers().some((marker: EventMarker): boolean => marker.button.isHovering()))
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
    this.pointer_position = screen;
    if (!this.risq.session.isReplaying()) {
      this.setHovering(false);
      return false;
    }
    if (this.drag_offset) {
      this.panel_position = { x: screen.x - this.drag_offset.x, y: screen.y - this.drag_offset.y };
    }
    this.layout(transform);
    if (this.drag_offset) {
      return true;
    }
    const panel_hit = super.mousemove(canvas, screen, transform);
    for (const button of [this.jump, this.next, this.close]) {
      if (this.panel_visible) {
        button.mousemove(canvas, screen, transform);
      } else {
        button.setHovering(false);
      }
    }
    let claimed = panel_hit;
    for (const { button } of this.visibleMarkers().reverse()) {
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
      const pressed = [this.jump, this.next, this.close].some((button: RisqEventButton): boolean =>
        button.mousedown(e)
      );
      if (!pressed && e.button === 0) {
        this.drag_offset = { x: this.pointer_position.x - this.xi(), y: this.pointer_position.y - this.yi() };
      }
      return true;
    }
    const marker = this.visibleMarkers()
      .reverse()
      .find((marker: EventMarker): boolean => marker.button.isHovering());
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
    if (this.drag_offset) {
      this.drag_offset = undefined;
      return;
    }
    for (const button of [
      this.jump,
      this.next,
      this.close,
      ...this.visibleMarkers().map((marker: EventMarker): RisqEventButton => marker.button),
    ]) {
      button.mouseup(e);
    }
  }
  override setClicking(clicking: boolean): void {
    super.setClicking(clicking);
    if (!clicking) {
      this.drag_offset = undefined;
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
    return (
      !!this.drag_offset ||
      [
        this.jump,
        this.next,
        this.close,
        ...this.visibleMarkers().map((marker: EventMarker): RisqEventButton => marker.button),
      ].some((button: RisqEventButton): boolean => button.isClicking())
    );
  }
}
