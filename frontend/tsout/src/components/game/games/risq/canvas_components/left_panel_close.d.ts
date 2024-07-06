import { DwgSquareButton } from '../../../util/canvas_components/button/square_button';
import { DwgRisq } from '../risq';
export declare class RisqLeftPanelButton extends DwgSquareButton {
    private risq;
    constructor(risq: DwgRisq);
    protected hovered(): void;
    protected unhovered(): void;
    protected clicked(): void;
    protected released(): void;
}
