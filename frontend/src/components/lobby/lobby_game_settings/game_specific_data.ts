import {capitalize} from "../../../scripts/util";
import {GameType} from "../data_models";

/** Returns a map of readable game settings */
export function getReadableGameSettings(settings: object, game_type: GameType):[string, string][] {
  const m: [string, string][] = [];
  const add = (s1: string, s2: any) => {
    m.push([capitalize(s1.replace('_', ' ')), s2.toString()]);
  };
  for (const [setting_name, setting] of Object.entries(settings)) {
    switch(game_type) {
      case GameType.FIDDLESTICKS:
        if (setting_name === 'ai_players') {
          const ai_players = setting as AiPlayerFiddlesticks[];
          add('AI Players', ai_players.length);
          for (const player of ai_players) {
            add('Theory Model 2', player.nickname);
          }
        } else {
          add(setting_name, setting);
        }
        break;
      default:
        add(setting_name, setting);
        break;
    }
  }
  return m;
}

/** Data describing fiddlesticks-specific game settings */
export declare interface GameSettingsFiddlesticks {
  min_round?: number;
  max_round?: number;
  round_points: number;
  trick_points: number;
  ai_players: AiPlayerFiddlesticks[];
}

/** Deta describing the settings for an ai player in fiddlesticks */
export declare interface AiPlayerFiddlesticks {
  nickname: string;
  //model: AiPlayerFiddlesticksModel;
}
