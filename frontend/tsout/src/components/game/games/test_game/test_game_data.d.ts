import { GameBase, GamePlayer } from '../../data_models';
export declare interface GameTestGame {
    game_base: GameBase;
    players: TestGamePlayer[];
}
export declare interface TestGamePlayer {
    player: GamePlayer;
}
