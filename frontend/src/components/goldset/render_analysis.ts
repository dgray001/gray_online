import type { GoldItem } from './types';
import { actionName, cardName, element, percent, table } from './dom';

export function renderAnalysis(host: HTMLElement, item: GoldItem): void {
  const a = item.analysis;
  host.replaceChildren(element('h2', 'Model4 reference'));
  host.append(
    element('p', `Snapshot captured ${new Date(item.created_at).toLocaleString()}. Labels retain this saved reference.`)
  );
  host.append(element('p', `Sampled source-game action: ${actionName(item.state, item.model4_action)}.`));
  host.append(element('p', a.assumption));
  if (item.state.kind === 'bet') {
    host.append(
      element(
        'p',
        `Model4 raw bid: ${a.raw_bid.toFixed(3)}. It clamps ${a.scale.toFixed(3)} × summed card contributions + ${a.optimism.toFixed(3)} to the legal range, then rounds probabilistically.`
      )
    );
    host.append(
      table(
        ['Bid', 'Model4 chooses', 'Estimated exact-bid chance', 'Expected round points'],
        (a.bid_probabilities ?? []).map((p: number, bid: number): string[] => [
          `${bid}`,
          percent(p),
          percent(a.trick_probabilities?.[bid] ?? 0),
          ((a.trick_probabilities?.[bid] ?? 0) * (item.state.round_points + item.state.trick_points * bid)).toFixed(3),
        ])
      )
    );
    if (item.state.hand_size === 1) {
      host.append(
        element(
          'p',
          `${a.beating_cards} of ${a.unknown_cards} unknown cards beat your card. Uniform sampling is without replacement; when you do not lead off-trump, the lead suit must also match.`
        )
      );
    }
    host.append(
      table(
        ['Card', 'Model win estimate', 'Bid contribution'],
        a.cards.map((c): string[] => [cardName(c), percent(c.win_probability), c.contribution.toFixed(4)])
      )
    );
  } else {
    host.append(
      element(
        'p',
        `Probability is proportional to exp(clamp(utility / ${Math.max(0.01, a.temperature).toFixed(3)} + heuristic − highest logit, −5, 0)).`
      )
    );
    host.append(
      table(
        ['Card', 'Win estimate', 'Contract utility', 'Heuristic', 'Choice chance'],
        a.cards.map((c): string[] => [
          cardName(c),
          percent(c.win_probability),
          c.utility.toFixed(4),
          c.heuristic.toFixed(4),
          percent(c.probability),
        ])
      )
    );
  }
  host.append(element('p', 'Round-point estimates are references; your label should pursue winning the game.'));
}
