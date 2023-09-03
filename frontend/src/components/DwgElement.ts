import {until} from "../scripts/util";

/** Describes an html element used by DwgElements */
interface DwgHtmlElement {
  selector: string;
  element: HTMLElement;
}

/** Not sure this actually does anything */
interface DwgTypedElement<T extends HTMLElement> {
  selector: string;
  element: T;
}

export abstract class DwgElement extends HTMLElement {
  fully_parsed = false;
  els: DwgHtmlElement[] = [];

  async connectedCallback() {
    await until(this.elementsParsed.bind(this));
  }

  elementsParsed(): boolean {
    let parsed = true;
    for (const el of this.els) {
      if (!el.element) {
        el.element = this.querySelector(el.selector);
      }
      if (!el.element) {
        parsed = false;
        continue;
      }
      if (el.element instanceof DwgElement) {
        if (!el.element.fully_parsed) {
          parsed = false;
        }
      }
    }
    this.fully_parsed = parsed;
    return this.fully_parsed;
  }

  setElement<T extends HTMLElement>(element: T, selector: string) {
    this.els.push({selector, element} as DwgTypedElement<T>);
  }
}

class testy extends DwgElement {
  test: HTMLButtonElement;

  constructor() {
    super();
    this.setElement(this.test, 'test');
  }
}