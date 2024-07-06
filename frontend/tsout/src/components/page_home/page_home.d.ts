import { DwgElement } from '../dwg_element';
import './page_home.scss';
import '../lobby/lobby';
import '../game/game';
import '../lobby/lobby_connector/lobby_connector';
export declare class DwgPageHome extends DwgElement {
    private lobby;
    private game;
    private lobby_connector;
    private client_on_mobile;
    constructor();
    protected parsedCallback(): void;
    private tryConnectionAgain;
}
