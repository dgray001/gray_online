import { GameType } from "../lobby/data_models";
import { DwgGame } from "./game";
export declare interface GameComponent {
    initialize(abstract_game: DwgGame, game: Game): Promise<void>;
    gameUpdate(update: UpdateMessage): Promise<void>;
}
export declare interface Game {
    game_base: GameBase;
    players: {
        player: GamePlayer;
    }[];
}
export declare interface UpdateMessage {
    update_id: number;
    kind: string;
    update: any;
}
export declare interface PlayerAction {
    action_id: number;
    client_id: number;
    kind: string;
    action: any;
}
export declare interface GameFromServer {
    game_base: GameBaseFromServer;
}
export declare function serverResponseToGame(gameFromServer: GameFromServer, client_id: number): Game;
export declare interface GameBaseFromServer {
    game_id: number;
    game_type: GameType;
    game_started: boolean;
    game_ended: boolean;
    players: GamePlayer[];
    viewers: GameViewer[];
    player_actions?: PlayerAction[];
    viewer_updates?: UpdateMessage[];
}
export declare interface GameBase {
    game_id: number;
    game_type: GameType;
    game_started: boolean;
    game_ended: boolean;
    players: Map<number, GamePlayer>;
    viewers: Map<number, GameViewer>;
    player_actions?: Map<number, PlayerAction>;
    updates?: Map<number, UpdateMessage>;
    last_continuous_update_id?: number;
}
export declare interface GamePlayer {
    client_id: number;
    player_id: number;
    nickname: string;
    connected: boolean;
    updates?: UpdateMessage[];
}
export declare interface GameViewer {
    client_id: number;
    nickname: string;
    connected: boolean;
    updates?: UpdateMessage[];
}
