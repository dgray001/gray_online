import { DialogSize } from '../../../../../dialog_box/dialog_box';
import { createMessage } from '../../../../../lobby/data_models';
import type { RisqRightPanel } from '../../canvas_components/right_panel/right_panel';
import type { RisqSession } from '../session';
import type { RisqOrdersModel } from './orders_model';
import type { RisqOrderPlanning } from './planning';

import '../../../../../dialog_box/confirm_dialog/confirm_dialog';

/** Submitting and unsubmitting this turn's pending orders, with one request in flight at a time */
export class RisqOrderSubmission {
  private toggling = false;
  private submitted_times = 0;

  constructor(
    private host: HTMLElement,
    private session: RisqSession,
    private orders_model: RisqOrdersModel,
    private planning: RisqOrderPlanning,
    private right_panel: RisqRightPanel
  ) {}

  submittedTimes(): number {
    return this.submitted_times;
  }

  turnStarted() {
    if (this.session.getPlayerId() > -1) {
      this.submitted_times = 0;
    }
  }

  submitted() {
    this.toggling = false;
    this.orders_model.clearPending();
  }

  unsubmitted() {
    this.toggling = false;
    this.orders_model.revertSubmittedToPending();
  }

  /** Submits directly, or first confirms when units/buildings are idle or resources would go negative */
  confirmSubmit() {
    const n_units = this.planning.idleUnitCount();
    const n_buildings = this.planning.idleBuildingCount();
    const n = n_units + n_buildings;
    const over_budget = this.planning.hasNegativeResources();
    if (n === 0 && !over_budget) {
      this.toggle();
      return;
    }
    const warnings: string[] = [];
    if (n > 0) {
      let noun: string;
      if (n_units > 0 && n_buildings > 0) {
        noun = 'units and buildings';
      } else if (n_buildings > 0) {
        noun = n_buildings === 1 ? 'building' : 'buildings';
      } else {
        noun = n_units === 1 ? 'unit' : 'units';
      }
      warnings.push(`You have ${n} idle ${noun}.`);
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

  toggle() {
    const player = this.session.getPlayer();
    if (this.toggling || !this.session.givingOrders() || !player) {
      this.toggling = false;
      return;
    }
    this.toggling = true;
    const sender = `player-${this.session.getPlayerId()}`;
    if (player.orders_submitted) {
      this.right_panel.unsubmittingOrders();
      this.dispatch(createMessage(sender, 'game-update', '', 'unsubmit-orders'));
    } else {
      this.right_panel.submittingOrders();
      const content = JSON.stringify({ orders: this.orders_model.pendingOrders() });
      this.dispatch(createMessage(sender, 'game-update', content, 'submit-orders'));
      this.submitted_times++;
    }
  }

  private dispatch(game_update: string) {
    this.host.dispatchEvent(new CustomEvent('game_update', { detail: game_update, bubbles: true }));
  }
}
