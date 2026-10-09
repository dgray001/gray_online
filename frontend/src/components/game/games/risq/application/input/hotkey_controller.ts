import type { ColorRGB } from '../../../../../../scripts/color_rgb';
import { getSettings } from '../../../../../../scripts/settings_store';
import { isDialogOpen, isTypingInInput } from '../../../../../../scripts/util';
import type { Point2D } from '../../../../util/objects2d';
import type { RisqLeftPanel } from '../../canvas_components/left_panel/left_panel';
import type { RisqViewport } from '../../rendering/board/viewport';
import { RisqViewMode } from '../../rendering/terrain';
import type { RisqCommands } from '../commands';
import type { RisqOrderSubmission } from '../orders/submission';
import type { RisqControlGroups } from '../selection/control_groups';
import type { RisqSelection } from '../selection/selection';
import type { RisqSession } from '../session';
import type { HotkeyCombo, RisqHotkeyLookupEntry } from './hotkeys';
import {
  RisqHotkeyAction,
  buildHotkeyLookup,
  comboFromKeyboardEvent,
  comboFromMouseEvent,
  comboToString,
  hotkeyApplies,
} from './hotkeys';

/** UI actions hotkeys trigger that live on the game component */
export declare interface HotkeyHost {
  showMessage(text: string, color: ColorRGB): void;
  reviewLastTurnReport(): void;
  openTechTree(): void;
  selectNextIdleUnit(): void;
  goToCoordinate(coordinate: Point2D): void;
}

const VIEW_MODE_ACTIONS: Partial<Record<RisqHotkeyAction, RisqViewMode>> = {
  [RisqHotkeyAction.VIEW_MODE_ALL]: RisqViewMode.ALL,
  [RisqHotkeyAction.VIEW_MODE_RESOURCE]: RisqViewMode.RESOURCE,
  [RisqHotkeyAction.VIEW_MODE_MILITARY]: RisqViewMode.MILITARY,
  [RisqHotkeyAction.VIEW_MODE_OWNERSHIP]: RisqViewMode.OWNERSHIP,
  [RisqHotkeyAction.VIEW_MODE_REGION]: RisqViewMode.REGION,
};

/** Keyboard and extra-mouse-button input: control groups plus configurable hotkeys, resolved against the selection */
export class RisqHotkeyController {
  private pressed_keys = new Map<string, RisqHotkeyLookupEntry>();
  private last_group_press: { group: number; time: number } | undefined;
  private input_observer = new MutationObserver(() => this.cancelBlockedInput());

  private clearPressed = (): void => {
    this.pressed_keys.clear();
    this.last_group_press = undefined;
    this.input_observer.disconnect();
  };

  private cancelBlockedInput = (): void => {
    if (isTypingInInput() || isDialogOpen()) {
      this.clearPressed();
    }
  };

  constructor(
    private host: HotkeyHost,
    private session: RisqSession,
    private viewport: RisqViewport,
    private left_panel: RisqLeftPanel,
    private selection: RisqSelection,
    private control_groups: RisqControlGroups,
    private commands: RisqCommands,
    private submission: RisqOrderSubmission
  ) {}

  attach(): void {
    document.body.addEventListener('keydown', this.handleKeydown);
    document.body.addEventListener('keyup', this.handleKeyup);
    document.body.addEventListener('focusin', this.cancelBlockedInput);
    window.addEventListener('blur', this.clearPressed);
  }

  detach(): void {
    document.body.removeEventListener('keydown', this.handleKeydown);
    document.body.removeEventListener('keyup', this.handleKeyup);
    document.body.removeEventListener('focusin', this.cancelBlockedInput);
    window.removeEventListener('blur', this.clearPressed);
    this.clearPressed();
  }

  private handleKeydown = (e: KeyboardEvent): void => {
    if (isTypingInInput() || isDialogOpen()) {
      this.clearPressed();
      return;
    }
    if (/^[0-9]$/.test(e.key)) {
      e.preventDefault();
      if (e.repeat) {
        return;
      }
      const group = e.key === '0' ? 10 : parseInt(e.key, 10);
      if (e.ctrlKey) {
        this.last_group_press = undefined;
        this.control_groups.assign(group);
      } else {
        const time = performance.now();
        const previous = this.last_group_press;
        const coordinate = this.control_groups.recall(group);
        const double_press = previous?.group === group && time - previous.time <= 300;
        this.last_group_press = double_press ? undefined : { group, time };
        if (double_press && coordinate) {
          this.host.goToCoordinate(coordinate);
        }
      }
      return;
    }
    this.last_group_press = undefined;
    const entry =
      this.pressed_keys.get(e.code) ?? (!e.repeat ? this.resolveEntry(comboFromKeyboardEvent(e)) : undefined);
    if (entry) {
      e.preventDefault();
      this.pressBinding(e.code, entry);
    }
  };

  private handleKeyup = (e: KeyboardEvent): void => {
    const entry = this.releaseBinding(e.code);
    if (isTypingInInput() || isDialogOpen()) {
      this.clearPressed();
      return;
    }
    if (entry) {
      e.preventDefault();
      this.released(entry);
    }
  };

  extraMouseButton(e: MouseEvent): void {
    if (isTypingInInput() || isDialogOpen()) {
      this.clearPressed();
      return;
    }
    const key = `mouse${e.button}`;
    if (e.type === 'mousedown') {
      const entry = this.resolveEntry(comboFromMouseEvent(e));
      if (entry) {
        e.preventDefault();
        this.pressBinding(key, entry);
      }
    } else {
      const entry = this.releaseBinding(key);
      if (entry) {
        e.preventDefault();
        this.released(entry);
      }
    }
  }

  private pressBinding(key: string, entry: RisqHotkeyLookupEntry): void {
    this.pressed_keys.set(key, entry);
    const observer_options: MutationObserverInit = { subtree: true };
    observer_options.childList = true;
    this.input_observer.observe(document.body, observer_options);
    this.pressed(entry);
  }

  private releaseBinding(key: string): RisqHotkeyLookupEntry | undefined {
    const entry = this.pressed_keys.get(key);
    this.pressed_keys.delete(key);
    if (this.pressed_keys.size === 0) {
      this.input_observer.disconnect();
    }
    return entry;
  }

  private resolveEntry(combo: HotkeyCombo): RisqHotkeyLookupEntry | undefined {
    const entries = buildHotkeyLookup(getSettings().risq_hotkeys).get(comboToString(combo)) ?? [];
    if (entries.length <= 1) {
      return entries[0];
    }
    const hotkey_selection = this.selection.hotkeySelection();
    return hotkey_selection ? entries.find((entry) => hotkeyApplies(entry, hotkey_selection)) : undefined;
  }

  private pressed(entry: RisqHotkeyLookupEntry | undefined): void {
    if (entry?.kind !== 'cycle_building') {
      return;
    }
    const next = this.selection.cycleBuildingOfType(entry.id);
    if (next) {
      this.host.goToCoordinate(next.space_coordinate);
    }
  }

  private released(entry: RisqHotkeyLookupEntry | undefined): void {
    if (!entry) {
      return;
    }
    if (entry.kind === 'action') {
      this.action(entry.action);
    } else if (entry.kind === 'create_unit' || entry.kind === 'research_tech' || entry.kind === 'build_building') {
      this.left_panel.triggerProducible(entry.kind, entry.id);
    } else if (entry.kind === 'hire_mercenary') {
      const mercenary = this.session.getPlayer()?.available_mercenaries.find((m) => m.id === entry.id);
      if (mercenary) {
        this.commands.toggleMercenary(mercenary);
      }
    }
  }

  private action(action: RisqHotkeyAction): void {
    const view_mode = VIEW_MODE_ACTIONS[action];
    if (view_mode !== undefined) {
      this.viewport.setViewMode(view_mode);
      return;
    }
    switch (action) {
      case RisqHotkeyAction.CYCLE_VIEW_MODE:
        this.viewport.cycleViewMode();
        break;
      case RisqHotkeyAction.SUMMARY_REPORT:
        this.host.reviewLastTurnReport();
        break;
      case RisqHotkeyAction.TECH_TREE:
        this.host.openTechTree();
        break;
      case RisqHotkeyAction.NEXT_IDLE:
        this.host.selectNextIdleUnit();
        break;
      case RisqHotkeyAction.GO_TO_SELECTION: {
        const coordinate = this.selection.coordinate();
        if (coordinate) {
          this.host.goToCoordinate(coordinate);
        }
        break;
      }
      case RisqHotkeyAction.SUBMIT_ORDERS:
        this.submission.confirmSubmit();
        break;
      default:
        this.left_panel.triggerAction(action);
        break;
    }
  }
}
