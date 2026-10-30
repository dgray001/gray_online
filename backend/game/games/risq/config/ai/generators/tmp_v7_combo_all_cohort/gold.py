from __future__ import annotations

from common import RESOURCES, Record, aggregate, branch, selection, variable


def mercenary_rates() -> list[Record]:
    actions = [variable(f"mercenary_rate_{resource}", 0) for resource in RESOURCES]
    for unit in (101, 102):
        producers = selection("buildings", dict(owner="own", state="complete", can_create=unit))
        actions.append(variable(f"native_producers_{unit}", aggregate("count", producers)))
        gap = f"max(0, min(var(goal_{unit}) - var(own_{unit}), gameVar(population_max) - gameVar(population)))"
        condition = f"(var(offer.id) == {unit}) * (var(native_producers_{unit}) == 0)"
        spending = [variable(f"mercenary_rate_{resource}",
                             f"var(mercenary_rate_{resource}) + ({gap}) * var(offer.cost_{resource}) / var(tune_planning_horizon)")
                    for resource in RESOURCES]
        actions.append(dict(action="for_each", source="mercenaries", **{"as": "offer"},
                            do=[branch(condition, spending)]))
    return actions


def gold_expansion() -> list[Record]:
    known_spaces = selection("spaces", dict(vision_at_least=1, **{"if": "var(item.gold_income) > 0"}))
    sites = selection("spaces", dict(owner="unowned", vision_at_least=1,
                                      near=dict(distance=1, of=selection("spaces", dict(owner="own"))),
                                      **{"if": "var(item.gold_income) > 0"}))
    site_income = aggregate("max", sites, "var(item.gold_income)")
    site_income["max"]["default"] = 0
    return [variable("map_space_gold", aggregate("sum", known_spaces, "var(item.gold_income)")),
            variable("known_gold_spaces", aggregate("count", known_spaces)),
            variable("gold_site_income", site_income),
            variable("gold_shortage", "max(0, var(rate_gold) + (var(research_budget_gold) - gameVar(gold)) / var(tune_planning_horizon) - var(gold_income))"),
            variable("want_21", "(var(tune_gold_expansion_weight) > 0) * (var(gold_site_income) > 0) * var(gold_shortage) / max(0.001, var(gold_site_income))")]
