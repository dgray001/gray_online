# V7 protection and funding tuning

## Canonical combined policy

Canonical v7 includes safe repair admission, selective sheltering, bounded local raid responses and queued-defender reserve credits.
The promoted 500-game eval has 438 favorable outcomes, four severe losses and 223 elimination wins with no elimination losses; zero game errors.
The generator output is byte-identical to the evaluated winner. Results and provenance are in `sim/outputs/eval/`; see the repository's `RISQ_AI_REFINEMENT.md`.
One selector-budget warning remains unattributed after 20 targeted debug replays. Temporary policy experiments have been removed.

## Historical protection experiments

Earlier v7 also included rate-aware gathering with a 1.25-turn window and two search passes.
Villager power is 0.58 before Farming and 0.84 after; civilian crush odds are 1.4.
Base military powers: blunt 1; piercing 1.5; heavy 3.3; siege 2.4.
Army calibration: 15,300 fights, 192 wall checks; held-out decisive accuracy 86%, versus previous estimates' 75%.
Own research selects fitted tech-state powers once; enemy techs remain estimated from buildings and range.
Catapults: base 2.4; Ballistics 2.5; Shrapnel 2.7; both 3.2. Escalades has no flat open-field power bonus.
At that stage, recruitment, interrupted-work resumption, worker allocation and construction logic were unchanged.
The 500-game results below are a pre-change baseline, not an evaluation of these additions.

Scope: limit protected demand and tune existing parameters; retain production-first worker allocation.
The only model structure added is villager, military-count and military-producer protection limits.
Default protection: villagers through their current goal; oldest one military producer; military below 20 units.
Parameters: protected_villager_share=1; protected_military_producers=1; protected_military_count=20.
Protection returns below the threshold. Total demand, production and building counts remain uncapped.

Limiting protection alone often leaves research starved by the investment-share and workforce-maturity cap.
Tune investment share, saving strength and planning horizons together with protection.
Shorter planning horizons also lower worker goals when stockpiles are large.
Larger workforce forecasts can delay upgrades; more workers or faster research are not monotonic improvements.
The military cutoff counts scouts and raiders, while main-army strength excludes their buckets.
Smaller raids retain more soldiers for main fights. Raw troop count and K/D do not determine outcomes alone.

Winning existing settings:
- investment_share=0.6; planning_horizon=6; construction_horizon=6; investment_horizon=20.
- additional_producer_margin=0.6; smith_saving_max=6; smith_urgency_turns=3.
- transfer_deadband=0; shortage_turns=4; transfer_fraction=0.5.
- raid_min=1; raid_max=2; raid_frac=0.1.

Search: 291 distinct non-baseline configurations; 119 tested through full 100-turn games.
Limits swept: villagers 0.6/0.8/1; military 10/20/30/50/unlimited; producers 0/1/2/unlimited.
The catalog records 5,544 game runs before the final 500-game evaluation, with zero failed trials.
Early screens used 12 balanced seeds, three per standard map. Final gates used all 160 standard games.
Primary gate: 95/160 favorable vs 56/160; confirmation: 89/160 vs 56/160.
Elimination wins: primary 50 vs 19; confirmation 45 vs 25.
Favorable includes turn-limit standings, not only elimination wins.

Confirmation averages, candidate/control:
- First smith: 31.36/46.55; Attack: 35.96/58.10; Armor: 39.68/66.57.
- Both upgrades by turn 40: 112/160 vs 0/160.
- Villagers at turns 20/30/40: 20.86/20.89; 30.62/30.69; 39.75/40.19.
- Blunts produced by turns 20/30/40: 5.08/4.60; 12.70/12.18; 21.09/23.46.
- VC idle through turn 40: 2.20%/0.47%; the upgrade timing has a small late-opening production cost.
The fastest profile reached both upgrades by 40 in 144/160 games, but scored only 63/160 favorable.

Validation: 18 tests pass; generated v7 exactly matches the confirmed candidate. No Go changes were made.
Repeated matched gates matter: identical-order debug runs showed variation in combat target ties.
Full 500-game eval: 270/500 favorable (54%); elimination wins/losses 143/37; zero errors.
Standard maps: 88/160; ring 12/40, hexagon 28/40, rectangle 25/40, triangle 23/40.
Weak buckets: risk_world variants 0/35; divide 5/20; divide_established 1/10.
Those historical trial outputs were superseded by the canonical `sim/outputs/eval/`.

Temporary policy trial outputs were removed when the evaluated combined policy became canonical v7.
V8 owns the unified marginal-benefit funding model; see ../v8/DESIGN.md.
