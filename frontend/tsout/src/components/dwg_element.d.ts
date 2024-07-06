import './dwg_element.scss';
export declare abstract class DwgElement extends HTMLElement {
    protected htmlString: string;
    fully_parsed: boolean;
    private elsMetadata;
    connectedCallback(): Promise<void>;
    private elementsParsed;
    protected parsedCallback(): void;
    protected configureElement(name: string, element_id?: string): void;
    protected configureElements(...names: string[]): void;
}
