import { DwgElement } from '../../dwg_element';
import { GameSettings, GameType } from '../data_models';
import './lobby_game_settings.scss';
export declare class DwgLobbyGameSettings extends DwgElement {
    game_settings_wrapper: HTMLDivElement;
    game_chooser: HTMLSelectElement;
    max_players_input: HTMLInputElement;
    max_viewers_input: HTMLInputElement;
    game_specific_settings: HTMLDivElement;
    room_description: HTMLTextAreaElement;
    save_button: HTMLButtonElement;
    game_specific_settings_els: Map<string, HTMLInputElement>;
    constructor();
    protected parsedCallback(): void;
    getGameSpecificHTML(game_type: GameType): HTMLElement[];
    private createNumberElement;
    private createRowElement;
    setSettings(settings: GameSettings, room_description: string): void;
    private setNumberSetting;
    clearSettings(): void;
    getSettings(): GameSettings;
    getDescription(): string;
}
