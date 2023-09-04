import {until} from "../scripts/util";

/** Describes an html element used by DwgElements */
interface ElementMetadata {
  element_id: string;
  name: string;
  found_element: boolean;
}

export abstract class DwgElement extends HTMLElement {
  protected htmlString: string;
  fully_parsed = false;
  private elsMetadata: ElementMetadata[] = [];

  async connectedCallback() {
    this.innerHTML = this.htmlString;
    await until(this.elementsParsed.bind(this));
    this.parsedCallback();
  }

  private elementsParsed(): boolean {
    let parsed = true;
    for (const elMetadata of this.elsMetadata) {
      let el: HTMLElement|null = null;
      if (!elMetadata.found_element) {
        el = this.querySelector(`#${elMetadata.element_id}`);
        // @ts-ignore -> set value in subclass
        this['test'] = el;
        elMetadata.found_element = true;
      }
      if (!elMetadata.found_element) {
        parsed = false;
        continue;
      }
      if (el instanceof DwgElement) {
        if (!el.fully_parsed) {
          parsed = false;
        }
      }
    }
    this.fully_parsed = parsed;
    return this.fully_parsed;
  }

  protected parsedCallback() {}

  protected configureElement(name: string, element_id: string) {
    this.elsMetadata.push({element_id, name} as ElementMetadata);
  }
}