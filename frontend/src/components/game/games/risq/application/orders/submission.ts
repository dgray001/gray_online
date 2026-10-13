import { DialogSize } from '../../../../../dialog_box/dialog_box';
import { createMessage } from '../../../../../lobby/data_models';
import type { RisqRightPanel } from '../../canvas_components/right_panel/right_panel';
import type { RisqSession } from '../session';
import type { RisqOrdersModel } from './orders_model';
import type { RisqOrderPlanning } from './planning';
import { idleCategory } from './idle_category';
import type { RisqAutoRenewQueue } from '../auto_renew_queue';
import type { RisqGatherPointQueue } from '../gather_point_queue';

import '../../../../../dialog_box/confirm_dialog/confirm_dialog';

/** Submitting and unsubmitting this turn's pending orders, with one request in flight at a time */
export class RisqOrderSubmission {
  private get toggling(): boolean {
    return this.session.isToggling();
  }
  private set toggling(value: boolean) {
    this.session.setToggling(value);
  }
  private submitted_times = 0;

  constructor(
    private host: HTMLElement,
    private session: RisqSession,
    private orders_model: RisqOrdersModel,
    private planning: RisqOrderPlanning,
    private right_panel: RisqRightPanel,
    private auto_renew: RisqAutoRenewQueue,
    private gather_points: RisqGatherPointQueue
  ) {}

  submittedTimes(): number {
    return this.submitted_times;
  }

  turnStarted(): void {
    this.toggling = false;
    if (this.session.getPlayerId() > -1) {
      this.submitted_times = 0;
    }
  }

  submitted(): void {
    this.toggling = false;
    this.orders_model.clearPending();
  }

  unsubmitted(): void {
    this.toggling = false;
    this.orders_model.revertSubmittedToPending();
  }

  /** Submits directly, or first confirms when units/buildings are idle or resources would go negative */
  confirmSubmit(): void {
    const player = this.session.getPlayer();
    if (this.toggling || !this.session.givingOrders() || !player) {
      return;
    }
    if (player.orders_submitted) {
      this.toggle();
      return;
    }
    const idle_orderables = this.planning.idleOrderables();
    const n = idle_orderables.length;
    const over_budget = this.planning.hasNegativeResources();
    if (n === 0 && !over_budget) {
      this.toggle();
      return;
    }
    const warnings: string[] = [];
    if (n > 0) {
      let n_villagers = 0;
      let n_military = 0;
      let n_production = 0;
      let n_other_buildings = 0;
      for (const entity of idle_orderables) {
        switch (idleCategory(entity)) {
          case 'economic':
            n_villagers++;
            break;
          case 'military':
            n_military++;
            break;
          case 'unit_production':
            n_production++;
            break;
          case 'other_buildings':
            n_other_buildings++;
            break;
        }
      }
      const idle_counts: string[] = [];
      if (n_villagers > 0) {
        idle_counts.push(`${n_villagers} idle villager${n_villagers === 1 ? '' : 's'}`);
      }
      if (n_military > 0) {
        idle_counts.push(`${n_military} idle military unit${n_military === 1 ? '' : 's'}`);
      }
      if (n_production > 0) {
        idle_counts.push(`${n_production} idle unit-producing building${n_production === 1 ? '' : 's'}`);
      }
      if (n_other_buildings > 0) {
        idle_counts.push(`${n_other_buildings} idle non-unit-producing building${n_other_buildings === 1 ? '' : 's'}`);
      }
      warnings.push(`You have ${idle_counts.join(', ')}.`);
    }
    if (over_budget) {
      warnings.push('One or more resources will go negative.');
    }
    const dialog = document.createElement('dwg-confirm-dialog');
    dialog.setData({
      question: `Are you sure you want to submit orders? ${warnings.join(' ')}`,
      size: DialogSize.SMALL,
    });
    dialog.addEventListener('confirmed', () => this.toggle());
    this.host.appendChild(dialog);
  }

  async toggle(): Promise<void> {
    const player = this.session.getPlayer();
    if (this.toggling || !this.session.givingOrders() || !player) {
      return;
    }
    this.toggling = true;
    if (!player.orders_submitted) {
      this.right_panel.submittingOrders();
      if (
        !(await Promise.all([this.auto_renew.drain(), this.gather_points.drain()])).every(
          (success: boolean): boolean => success
        ) ||
        !this.session.givingOrders() ||
        this.session.getPlayer()?.orders_submitted
      ) {
        this.toggling = false;
        this.orders_model.triggerChange();
        return;
      }
    }
    const sender = `player-${this.session.getPlayerId()}`;
    if (player.orders_submitted) {
      this.right_panel.unsubmittingOrders();
      this.dispatch(createMessage(sender, 'game-update', '', 'unsubmit-orders'));
    } else {
      const content = JSON.stringify({ orders: this.orders_model.pendingOrders() });
      this.dispatch(createMessage(sender, 'game-update', content, 'submit-orders'));
      this.submitted_times++;
    }
  }

  private dispatch(game_update: string): void {
    this.host.dispatchEvent(new CustomEvent('game_update', { detail: game_update, bubbles: true }));
  }
}
