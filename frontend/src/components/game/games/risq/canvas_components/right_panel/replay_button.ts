import type { DwgRisq } from '../../risq';
import { RisqActionButton } from '../left_panel/actions/action_button';
import { RISQ_MESSAGE_WARNING_COLOR } from '../message_queue';

export class RisqReplayButton extends RisqActionButton {
  constructor(private risq: DwgRisq) {
    super(
      { row: 0, col: 0, image_path: 'icons/refresh_black32', description: 'Replay last turn', image_on_top: true },
      30
    );
  }

  override isClicking(): boolean {
    return super.isClicking() || this.risq.session.isReplaying();
  }

  override dataRefreshed(): void {
    this.dimmed = !this.risq.getGame()?.last_turn_journal && !this.risq.session.isReplaying();
  }

  protected released(): void {
    if (!this.isHovering()) {
      return;
    }
    if (this.dimmed) {
      this.risq.showMessage('No previous turn to replay', RISQ_MESSAGE_WARNING_COLOR);
    } else {
      this.risq.toggleReplay();
    }
  }
}
