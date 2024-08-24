import { DwgElement } from '../../../dwg_element';
import { GameComponent, UpdateMessage } from '../../data_models';
import { DwgGame } from '../../game';
import { GameTestGame } from './test_game_data';
import './test_game.scss';
export declare class DwgTestGame extends DwgElement implements GameComponent {
    private show_info;
    private end_game;
    private game;
    private player_id;
    constructor();
    protected parsedCallback(): void;
    initialize(abstract_game: DwgGame, game: GameTestGame): Promise<void>;
    gameUpdate(update: UpdateMessage): Promise<void>;
}
declare global {
    interface HTMLElementTagNameMap {
        'dwg-test-game': DwgTestGame;
    }
}
