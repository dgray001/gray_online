import type { ColorRGB } from '../../../../../../../scripts/color_rgb';
import { drawLine, drawText } from '../../../../../util/canvas_util';
import type { DwgRisq } from '../../../risq';
import type { LeftPanelLayout, PanelFrame } from '../layout';
import type { RisqStatsView } from './stats_view';

/** Everything a left-panel content renderer needs for one draw */
export declare interface PanelDrawContext {
  ctx: CanvasRenderingContext2D;
  risq: DwgRisq;
  frame: PanelFrame;
  layout: LeftPanelLayout;
  visibility: number;
  stats: RisqStatsView;
}

/** Draws the centered title; returns its height */
export function drawName(pc: PanelDrawContext, name: string): number {
  const text_size = Math.min(40, (1 / 12) * pc.frame.h());
  drawText(pc.ctx, name, {
    p: { x: pc.frame.xc(), y: pc.frame.yi() + 3 },
    w: pc.frame.w(),
    fill_style: 'black',
    align: 'center',
    font: `bold ${0.85 * text_size}px serif`,
  });
  return text_size + 3;
}

/** Draws a centered subtitle at yi; returns its height */
export function drawSubtitle(pc: PanelDrawContext, text: string, yi: number): number {
  drawText(pc.ctx, text, {
    p: { x: pc.frame.xc(), y: yi },
    w: pc.frame.w(),
    fill_style: 'black',
    align: 'center',
    font: '18px serif',
  });
  return 26;
}

/** Draws the header image filling the rest of the panel's top quarter; returns the height it reserved */
export function drawHeaderImage(
  pc: PanelDrawContext,
  yi: number,
  img: string | CanvasImageSource,
  color?: ColorRGB
): number {
  pc.ctx.beginPath();
  const max_img_height = 0.25 * pc.frame.h() - yi + pc.frame.yi();
  const img_height = Math.min(max_img_height - 6, 0.8 * pc.frame.w());
  const icon = typeof img !== 'string' ? img : color ? pc.risq.getPlayerColoredIcon(img, color) : pc.risq.getIcon(img);
  pc.ctx.drawImage(icon, 0.5 * (pc.frame.w() - img_height), yi, img_height, img_height);
  return max_img_height;
}

export function drawSeparator(pc: PanelDrawContext, yi: number) {
  pc.ctx.strokeStyle = 'rgba(60, 60, 60, 0.7)';
  pc.ctx.lineWidth = 2;
  drawLine(pc.ctx, { x: pc.frame.xi() + 0.1 * pc.frame.w(), y: yi }, { x: pc.frame.xf() - 0.1 * pc.frame.w(), y: yi });
}

/** Separators framing the garrison and action-grid bands, drawn only for the local player's selections */
export function drawOwnerSeparators(pc: PanelDrawContext, owner_player_id: number | undefined): boolean {
  if (owner_player_id === undefined || pc.risq.getPlayer()?.player.player_id !== owner_player_id) {
    return false;
  }
  drawSeparator(pc, pc.layout.separator_below_stats);
  if (pc.layout.garrison_rows > 0) {
    drawSeparator(pc, pc.layout.garrison_separator);
  }
  drawSeparator(pc, pc.layout.grid_bottom_separator);
  return true;
}
