from __future__ import annotations

from common import RESOURCES, Record, aggregate, branch, selection, store, variable, whole


def worker_groups() -> list[Record]:
    workers = selection("units", dict(owner="own", unit_ids=[1], not_in_set="construction_workers", **{
        "in": dict(order_kind=["none", "gather"]), "if": "1 - var(target.is_garrisoned)"}))
    actions = [store("economy_workers", workers)]
    category = aggregate("max", selection("gatherables", {"if": "var(item.uid) == var(target.order_target_uid)"}), "var(item.category)")
    category["max"]["default"] = -1
    for index, resource in enumerate(RESOURCES):
        actions.append(dict(action="select", pick=selection("units", dict(bucket=f"gather_{resource}")),
                            into=dict(bucket=f"gather_{resource}", mode="remove")))
        actions.append(store(f"nodes_{resource}", selection("gatherables", dict(category=resource))))
        actions.append(store(f"current_{resource}", selection("units", dict(in_set="economy_workers", **{
            "if": f"var(target.current_category) == {index}"}), **{"let": dict(current_category=category)})))
        actions.append(variable(f"shortage_{resource}", f"(var(workers_{resource}) > var(n_current_{resource})) * ((var(workers_{resource}) - var(n_current_{resource}) > var(tune_transfer_deadband)) + (gameVar({resource}) < var(need_{resource}) * var(tune_shortage_turns)) > 0)"))
    return actions


def bucket_size(resource: str) -> Record:
    return aggregate("count", selection("units", dict(owner="own", bucket=f"gather_{resource}")))


def fill(resource: str, amount: str) -> list[Record]:
    workers = selection("units", dict(in_set="economy_workers", bucket="none"),
                        rank=["var(target.order_kind) == -1", "0 - var(target.uid)"], limit=whole(amount))
    return [variable(f"gather_size_{resource}", bucket_size(resource)),
            dict(action="select", pick=workers, into=dict(bucket=f"gather_{resource}", mode="add"))]


def allocate_workers() -> list[Record]:
    actions: list[Record] = []
    for resource in RESOURCES:
        other_shortages = " + ".join(f"var(shortage_{other})" for other in RESOURCES if other != resource)
        release = f"min(max(0, var(n_current_{resource}) - var(workers_{resource})), max(1, {whole(f'var(n_current_{resource}) * var(tune_transfer_fraction)')}))"
        actions.append(variable(f"released_{resource}", f"({other_shortages} > 0) * (var(tune_transfer_fraction) > 0) * ({release})"))
        actions.append(dict(action="select", pick=selection("units", dict(in_set=f"current_{resource}"),
            rank="0 - var(target.uid)", limit=f"var(n_current_{resource}) - var(released_{resource})"),
            into=dict(bucket=f"gather_{resource}", mode="replace")))
    for resource in RESOURCES:
        actions.append(branch(f"var(shortage_{resource})", fill(resource, f"var(workers_{resource}) - var(gather_size_{resource})")))
    for resource in RESOURCES:
        actions.extend(fill(resource, f"var(workers_{resource}) - var(gather_size_{resource})"))
    for resource in RESOURCES:
        actions.extend(fill(resource, f"var(capacity_{resource}) - var(gather_size_{resource})"))
    actions.append(variable("food_workers_last", bucket_size("food"), True))
    return actions


def assign_nodes(resource: str, search_pass: int) -> Record:
    used = aggregate("count", selection("units", dict(owner="own", **{"in": dict(order_kind=["gather"])},
        **{"if": "var(item.order_target_uid) == var(target.uid)"})))
    staying = aggregate("count", selection("gatherables", dict(in_set=f"nodes_{resource}",
        **{"if": "var(item.uid) == var(actor.order_target_uid)"})))
    actors = selection("units", dict(bucket=f"gather_{resource}", **{"in": dict(order_kind=["none", "gather"])},
        **{"if": ["distance(actor, group) == 0", "var(actor.staying) == 0"]}),
        **{"let": dict(staying=staying)})
    live_capacity = "var(target.gather_capacity) - var(target.workers)"
    targets = selection("gatherables", dict(in_set=f"nodes_{resource}"), **{"let": dict(workers=used)},
        rank=["0 - distance(group, target)", "0 - var(target.uid)"],
        limit="var(gather_node_limit)", skip=f"{search_pass} * var(gather_node_limit)",
        capacity=live_capacity)
    score = "0 - var(tune_gather_distance_weight) * distance(actor, target) - var(tune_gather_depletion_penalty) * (var(target.amount_left) <= var(target.workers) * var(target.gather_speed))"
    return dict(action="select", actors=actors, pick=targets, match=dict(score=score), issue=[dict(gather={})])


def gathering(parameters: dict[str, float]) -> list[Record]:
    actions: list[Record] = []
    for resource in RESOURCES:
        members = selection("units", dict(bucket=f"gather_{resource}"))
        groups = selection("spaces", dict(near=dict(distance=0, of=members)))
        count = aggregate("count", selection("units", dict(bucket=f"gather_{resource}", **{
            "if": "(var(item.x) == var(group.x)) * (var(item.y) == var(group.y))"})))
        actions.append(dict(action="for_each", select=groups, **{"as": "group"}, do=[
            variable("gather_group_workers", count),
            variable("gather_node_limit", whole("var(tune_gather_nodes_min) + var(tune_gather_nodes_per_worker) * var(gather_group_workers)")),
            *[assign_nodes(resource, search_pass) for search_pass in range(int(parameters["gather_search_passes"]))]]))
    return actions
