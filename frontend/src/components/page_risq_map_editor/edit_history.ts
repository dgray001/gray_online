import type { MapDoc } from './map_doc';

export interface EditCommand {
  apply: (doc: MapDoc) => void;
  revert: (doc: MapDoc) => void;
}

export class EditHistory {
  private undo_stack: EditCommand[] = [];
  private redo_stack: EditCommand[] = [];

  record(command: EditCommand): void {
    this.undo_stack.push(command);
    this.redo_stack = [];
  }

  undo(doc: MapDoc): boolean {
    const command = this.undo_stack.pop();
    if (!command) {
      return false;
    }
    command.revert(doc);
    this.redo_stack.push(command);
    return true;
  }

  redo(doc: MapDoc): boolean {
    const command = this.redo_stack.pop();
    if (!command) {
      return false;
    }
    command.apply(doc);
    this.undo_stack.push(command);
    return true;
  }

  clear(): void {
    this.undo_stack = [];
    this.redo_stack = [];
  }
}
