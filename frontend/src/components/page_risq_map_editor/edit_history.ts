import type { MapDoc } from './map_doc';

export interface EditCommand {
  apply: (doc: MapDoc) => void;
  revert: (doc: MapDoc) => void;
}

export class EditHistory {
  private undo_stack: EditCommand[] = [];
  private redo_stack: EditCommand[] = [];

  private saved_depth = 0;
  private revision_count = 0;

  revision(): number {
    return this.revision_count;
  }

  record(command: EditCommand): void {
    this.revision_count++;
    if (this.undo_stack.length < this.saved_depth) {
      this.saved_depth = -1;
    }
    this.undo_stack.push(command);
    this.redo_stack = [];
  }

  markSaved(): void {
    this.saved_depth = this.undo_stack.length;
  }

  isDirty(): boolean {
    return this.undo_stack.length !== this.saved_depth;
  }

  undo(doc: MapDoc): boolean {
    const command = this.undo_stack.pop();
    if (!command) {
      return false;
    }
    command.revert(doc);
    this.revision_count++;
    this.redo_stack.push(command);
    return true;
  }

  redo(doc: MapDoc): boolean {
    const command = this.redo_stack.pop();
    if (!command) {
      return false;
    }
    command.apply(doc);
    this.revision_count++;
    this.undo_stack.push(command);
    return true;
  }

  clear(): void {
    this.revision_count++;
    this.undo_stack = [];
    this.redo_stack = [];
    this.saved_depth = 0;
  }
}
