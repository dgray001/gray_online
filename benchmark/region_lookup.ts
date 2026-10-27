import { resolveSelectionData } from '../frontend/src/components/game/games/risq/canvas_components/left_panel/selection_queries';
import { RisqSession } from '../frontend/src/components/game/games/risq/application/session';
import { LeftPanelDataType } from '../frontend/src/components/game/games/risq/canvas_components/left_panel/left_panel_data';
import { GameRisq, RisqRegion } from '../frontend/src/components/game/games/risq/model/types';
import { serverToGameRisq } from '../frontend/src/components/game/games/risq/transport/snapshot_conversion';
import { GameRisqFromServer } from '../frontend/src/components/game/games/risq/transport/snapshot_types';

function createDummyGame(numRegions: number): GameRisqFromServer {
  const regions: any[] = [];
  for (let i = 0; i < numRegions; i++) {
    regions.push({
      name: `Region_${i}`,
      spaces: [],
      // other region properties...
    });
  }
  return {
    game_base: { game_ended: false } as any,
    turn_number: 1,
    giving_orders: true,
    players: [],
    spaces: [],
    regions: regions,
  } as unknown as GameRisqFromServer;
}

function runBenchmark() {
  const session = new RisqSession();

  // Create 10000 regions
  const dummyGame = createDummyGame(10000);
  session.replaceSnapshot(dummyGame);

  const iterations = 10000;
  let matches = 0;

  // Warmup
  for (let i = 0; i < 1000; i++) {
      const targetRegionName = `Region_${i}`;
      resolveSelectionData(
        { data_type: LeftPanelDataType.REGION, data: { name: targetRegionName } } as any,
        session
      );
  }

  const start = performance.now();
  for (let i = 0; i < iterations; i++) {
    const targetRegionName = `Region_${i}`;
    const result = resolveSelectionData(
      { data_type: LeftPanelDataType.REGION, data: { name: targetRegionName } } as any,
      session
    );
    if (result) matches++;
  }
  const end = performance.now();

  console.log(`Matched ${matches} regions.`);
  console.log(`Time taken: ${(end - start).toFixed(2)} ms for ${iterations} iterations`);
}

runBenchmark();
