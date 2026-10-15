import type { GameRisq } from './types';

/** Whether economic units can exist in the game; false hides every economy-specific display */
export function hasEconomy(game: GameRisq | undefined): boolean {
  return game?.has_economy !== false;
}

/** Whether buildings can exist in the game; false hides every building-specific display */
export function hasBuildings(game: GameRisq | undefined): boolean {
  return game?.has_buildings !== false;
}

/** A game with neither economy nor buildings, where only the military and ownership views make sense */
export function isMilitaryOnly(game: GameRisq | undefined): boolean {
  return !hasEconomy(game) && !hasBuildings(game);
}
