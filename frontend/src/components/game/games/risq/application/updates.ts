import type { UpdateMessage } from '../../../data_models';
import { messageDialog } from '../../../game';
import { perfEvent } from '../../../../../scripts/perf';
import { getSettings } from '../../../../../scripts/settings_store';
import { err, log } from '../../../../../scripts/log';
import type { RisqPlayer } from '../model/types';
import type {
  AutoRenewSetData,
  BuildingBehaviorSetData,
  DefaultUnitBehaviorSetData,
  GameRisqFromServer,
  GatherPointSetData,
  StartTurnData,
  SubmissionNoticeData,
  UnitBehaviorSetData,
} from '../transport/snapshot_types';
import type { RisqTurnReport } from '../transport/turn_report';
import { RISQ_MESSAGE_WARNING_COLOR } from '../canvas_components/message_queue';
import type { DwgRisq } from '../risq';
import { RisqHotkeyAction } from './input/hotkeys';
import {
  patchAutoRenew,
  patchBuildingBehavior,
  patchDefaultUnitBehavior,
  patchGatherPoint,
  patchUnitBehavior,
} from './setting_patches';

import '../dialogs/turn_report_dialog/turn_report_dialog';

/** Applies server updates to the game and surfaces turn-level results (turn report, outcome) */
export class RisqUpdates {
  private outcome_shown = false;

  constructor(private risq: DwgRisq) {}

  reset(): void {
    this.outcome_shown = false;
  }

  async gameUpdate(update: UpdateMessage): Promise<void> {
    try {
      switch (update.kind) {
        case 'start-turn':
          this.risq.commands.auto_renew.reset();
          this.risq.commands.gather_points.reset();
          this.applyStartTurn(update.content as StartTurnData);
          break;
        case 'resync':
          this.setNewGameData((update.content as StartTurnData).game);
          break;
        case 'submitted-orders':
        case 'unsubmitted-orders':
          this.applySubmission(update.content as SubmissionNoticeData);
          break;
        case 'unit-behavior-set':
          this.applySetting(() => patchUnitBehavior(this.risq.session, update.content as UnitBehaviorSetData));
          break;
        case 'building-behavior-set':
          this.applySetting(() => patchBuildingBehavior(this.risq.session, update.content as BuildingBehaviorSetData));
          break;
        case 'default-unit-behavior-set':
        case 'default-unit-stance-set':
          this.applySetting(() =>
            patchDefaultUnitBehavior(this.risq.session, update.content as DefaultUnitBehaviorSetData)
          );
          break;
        case 'gather-point-set':
          this.applySetting(() => patchGatherPoint(this.risq.session, update.content as GatherPointSetData));
          this.risq.commands.gather_points.acknowledge((update.content as GatherPointSetData).building_id);
          break;
        case 'set-gather-point-failed':
          this.risq.commands.gather_points.acknowledge(undefined, false);
          this.risq.showMessage((update.content as { message: string }).message, RISQ_MESSAGE_WARNING_COLOR);
          break;
        case 'auto-renew-set':
          this.applySetting(() => patchAutoRenew(this.risq.session, update.content as AutoRenewSetData));
          this.risq.commands.auto_renew.acknowledge();
          break;
        case 'set-auto-renew-failed':
          this.risq.commands.auto_renew.acknowledge(false);
          this.risq.showMessage((update.content as { message: string }).message, RISQ_MESSAGE_WARNING_COLOR);
          this.risq.refreshPanels();
          break;
        case 'set-default-unit-behavior-failed':
          this.risq.showMessage((update.content as { message: string }).message, RISQ_MESSAGE_WARNING_COLOR);
          break;
        default:
          log(`Unknown game update type ${update.kind}`);
          break;
      }
    } catch (e) {
      err(`Error during game update ${JSON.stringify(update)}: ${e}`);
    }
  }

  private applySetting(patch: () => void): void {
    patch();
    this.risq.refreshPanels();
  }

  setNewGameData(new_game: GameRisqFromServer): void {
    const apply_start = performance.now();
    this.risq.session.replaceSnapshot(new_game);
    this.risq.board.setDefaultRotation(this.risq.session.getGame()?.rotation ?? 0);
    const convert_ms = performance.now() - apply_start;
    if (!this.risq.session.givingOrders() || this.risq.getPlayer()?.orders_submitted) {
      this.risq.commands.gather_points.reset();
    }
    const shell_game = this.risq.abstract_game?.getGame();
    if (shell_game) {
      shell_game.game_base.game_ended = new_game.game_base.game_ended;
    }
    if (new_game.game_base.game_ended) {
      this.risq.commands.auto_renew.reset();
      this.risq.armed.disarmAll();
      this.risq.orders_model.clearUnreceived();
    }
    this.risq.hover.forget();
    const raw_player = new_game.players[this.risq.getPlayerId()];
    this.risq.orders_model.setReceived(raw_player?.active_orders ?? []);
    if (raw_player?.orders_submitted) {
      this.risq.orders_model.setUnreceived(raw_player.unreceived_orders);
    }
    this.risq.session.setLastTurnReport(this.risq.getPlayer()?.turn_report ?? this.risq.session.getLastTurnReport());
    this.risq.refreshPanels();
    this.showGameOutcome();
    this.risq.commands.gather_points.reconcile();
    perfEvent('snapshot_apply', {
      turn: new_game.turn_number,
      convert_ms,
      total_ms: performance.now() - apply_start,
    });
  }

  private showGameOutcome(): void {
    const game = this.risq.getGame();
    if (!game?.game_base.game_ended || !game.outcome || this.outcome_shown) {
      return;
    }
    this.outcome_shown = true;
    const winners = game.outcome.winner_player_ids.map(
      (id: number): string =>
        game.players.find((player: RisqPlayer): boolean => player.player.player_id === id)!.player.nickname
    );
    const message =
      winners.length === 0
        ? 'No players remaining'
        : `${winners.length > 1 ? 'The winners are: ' : 'The winner is: '}${winners.join(', ')}`;
    messageDialog.call(this.risq, { message });
  }

  private applyStartTurn(data: StartTurnData): void {
    if (this.risq.session.isReplaying()) {
      this.risq.session.setReplaying(false);
      this.risq.replay_panel.close();
      this.risq.showMessage('Replay mode exited: new turn started', RISQ_MESSAGE_WARNING_COLOR);
    }
    this.risq.submission.turnStarted();
    this.risq.armed.disarmAll();
    this.risq.orders_model.setUnreceived([]); // last turn's batch is now in active_orders
    this.setNewGameData(data.game);
    const player = this.risq.getPlayer();
    if (
      !data.game.game_base.game_ended &&
      this.risq.getPlayerId() > -1 &&
      player?.turn_report &&
      getSettings().risq_display.auto_open_turn_report
    ) {
      this.openTurnReportDialog(player, player.turn_report);
    }
  }

  private applySubmission(data: SubmissionNoticeData): void {
    if (!this.risq.session.applySubmission(data)) {
      return;
    }
    if (data.player_id === this.risq.getPlayerId()) {
      if (data.orders_submitted) {
        this.risq.orders_model.setUnreceived(data.unreceived_orders ?? []);
      }
      this.risq.submission.acknowledged();
      this.risq.hover.forget();
    }
    const player = this.risq.getPlayer();
    const unsubmitted = this.risq
      .getGame()
      ?.players.filter((candidate) => !candidate.eliminated && !candidate.orders_submitted);
    if (!data.giving_orders) {
      this.risq.showMessage('Resolving turn...', RISQ_MESSAGE_WARNING_COLOR);
    } else if (player && !player.orders_submitted && unsubmitted?.length === 1) {
      this.risq.showMessage('All other players submitted orders', RISQ_MESSAGE_WARNING_COLOR);
    }
    if (!this.risq.session.givingOrders() || this.risq.getPlayer()?.orders_submitted) {
      this.risq.commands.gather_points.reset();
    }
    this.risq.refreshPanels();
  }

  reviewLastTurnReport(): void {
    const player = this.risq.getPlayer();
    const report = this.risq.session.getLastTurnReport();
    if (!report) {
      this.risq.showMessage('No previous turn report to review', RISQ_MESSAGE_WARNING_COLOR);
    } else if (player) {
      this.openTurnReportDialog(player, report);
    }
  }

  openTurnReportDialog(player: RisqPlayer, report: RisqTurnReport): void {
    const dialog = document.createElement('dwg-risq-turn-report-dialog');
    dialog.setData({
      risq: this.risq,
      player,
      report,
      close_hotkey: getSettings().risq_hotkeys.actions[RisqHotkeyAction.SUMMARY_REPORT],
    });
    this.risq.appendChild(dialog);
  }
}
