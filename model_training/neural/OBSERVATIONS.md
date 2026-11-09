# Observation schema 2

The current observation has 740 inputs. The original 370 inputs remain unchanged
as its prefix. Existing checkpoints use that original prefix automatically.

| Indices | Inputs |
|---|---|
| 0–369 | Original hand, seen cards, trick, trump suit, scores, bids, tricks, scoring |
| 370–525 | Per-card actor as relative-seat sine/cosine, and play order within the round |
| 526–565 | Known void suits for up to ten players, relative to the acting player |
| 566–645 | Eight current-game public profile values for each player |
| 646–725 | Eight prior-history profile values for each player; zero until supplied |
| 726–739 | Game progress, remaining rounds/points, relative standings, cards/tricks needed, trump rank, trick position |

Profiles encode mean bid fraction, exact-bid rate, mean trick fraction, trump use,
trump leads, face-card use, and observation counts. Raw public history retains
the acting seat, round, hand size, trick number, and play order for every action.
Progress explicitly distinguishes ascending/descending rounds and repetitions.
Prior profiles can be supplied through `SetPriorProfiles`; persistent player
history aggregation is separate from this initial annotation tool.
