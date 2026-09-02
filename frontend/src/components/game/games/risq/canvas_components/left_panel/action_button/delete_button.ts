import { RisqActionButton } from './action_button';

export class RisqDeleteButton extends RisqActionButton {
  protected released(): void {
    if (this.isHovering()) {
      // TODO: open a confirm dialog, then issue an OrderType_UnitDelete order on confirm
    }
  }
}
