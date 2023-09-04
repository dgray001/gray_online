export declare abstract class DwgElement extends HTMLElement {
    fully_parsed: boolean;
    private els;
    connectedCallback(): Promise<void>;
    private elementsParsed;
    protected setElement<T extends HTMLElement>(element: T, selector: string): void;
}
