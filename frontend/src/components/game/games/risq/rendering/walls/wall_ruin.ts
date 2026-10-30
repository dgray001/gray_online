import type { Breach } from './wall_breach';
import type { WallDamage } from './wall_damage';
import {
  breachFloor,
  breachSpecs,
  floorDepth,
  isRuin,
  isWeathered,
  rubbleStones,
  ruinFloor,
  ruinStones,
  seededRandom,
} from './wall_damage';

const PILE_STONES = [8, 8, 9, 12, 12];
const WEATHERED_WEAR = 0.55;
const RUIN_STONE_COUNT = 44;
const WEATHERED_STONE_COUNT = 24;
const RUIN_FLOOR_ROCKS = 26;
const WEATHERED_FLOOR_ROCKS = 12;

export function floorRockCount(damage: WallDamage): number {
  return isWeathered(damage) ? WEATHERED_FLOOR_ROCKS : isRuin(damage) ? RUIN_FLOOR_ROCKS : 6;
}

export function buildBreaches(damage: WallDamage, face: number, wall_h: number, unscaled_h: number): Breach[] {
  const seed = `face${face}|${damage}`;
  const stones = isWeathered(damage) ? WEATHERED_STONE_COUNT : RUIN_STONE_COUNT;
  const wear = isWeathered(damage) ? WEATHERED_WEAR : 1;
  return breachSpecs(damage, face, seededRandom(`${seed}|range`)).map((spec, i) => {
    const tag = i === 0 ? '' : String(i);
    const floor_random = seededRandom(`${seed}|floor${tag}`);
    const rubble_random = seededRandom(`${seed}|rubble${tag}`);
    return {
      spec,
      floor: isRuin(damage)
        ? ruinFloor(wall_h, floor_random, wear)
        : breachFloor(spec.range, wall_h, floorDepth(damage, spec.big), floor_random),
      stones: isRuin(damage)
        ? ruinStones(unscaled_h, stones, rubble_random)
        : rubbleStones(spec.range, unscaled_h, PILE_STONES[damage], rubble_random),
      seed: `${seed}${tag}`,
    };
  });
}
