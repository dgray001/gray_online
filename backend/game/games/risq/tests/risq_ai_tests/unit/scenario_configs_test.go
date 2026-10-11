package unit

const economyScenario = `{"rules":[
{"when":{"always":null},"then":[{"action":"set_var","name":"visits","value":"var(visits)+1","persist":true}]},
{"when":{"turn_equals":{"amount":1}},"then":[
{"action":"add_q","type":"technology","id":3,"weight":7},
{"action":"add_q","type":"unit","id":2,"weight":5},
{"action":"add_q","type":"building","id":2,"weight":3},
{"action":"produce","prioritize":true},{"action":"produce","prioritize":true},{"action":"buildNextInQ","prioritize":true,"max":1},
{"action":"create","unit_id":2,"building_ids":[2],"queue":2},
{"action":"hire","unit_id":2,"max":1},{"action":"create","unit_id":2,"building_ids":[2],"queue":3},
{"action":"set_bucket","bucket":"workers","size":2,"task":{"action":"gather","category":"wood"}},
{"action":"fill_bucket","bucket":"workers","unit_ids":[1],"eligible":["build"]},
{"action":"run_bucket","bucket":"workers"},{"action":"run_bucket","bucket":"workers"},
{"action":"move","unit_ids":[99],"targets":"coordinate","coordinate":{"x":"population+100*var(visits)","y":"wood*100+food"}}
]},
{"when":{"turn_equals":{"amount":2}},"then":[
{"action":"run_bucket","bucket":"workers","when_full":true},
{"action":"move","unit_ids":[99],"targets":"coordinate","coordinate":{"x":"population+100*var(visits)","y":"wood*100+food"}}
]},
{"when":{"turn_equals":{"amount":3}},"then":[
{"action":"repair","building_ids":[1]},{"action":"repair","building_ids":[1]},
{"action":"renew","building_ids":[2]},{"action":"renew","building_ids":[2]},
{"action":"garrison","unit_ids":[2],"building_ids":[1],"max":1},
{"action":"garrison","unit_ids":[2],"building_ids":[1],"max":1}
]},
{"when":{"turn_equals":{"amount":4}},"then":[
{"action":"building_stop","orders":["create"],"item_ids":[2],"max":1},
{"action":"building_stop","orders":["create"],"item_ids":[2],"max":1},
{"action":"create","unit_id":2,"building_ids":[2]},
{"action":"renew","building_ids":[1]},{"action":"renew","building_ids":[1]},
{"action":"move","unit_ids":[99],"targets":"coordinate","coordinate":{"x":"food","y":"wood"}}
]},
{"when":{"turn_equals":{"amount":5}},"then":[
{"action":"add_q","type":"resource","cost":{"food":100},"weight":1},
{"action":"gather","unit_ids":[1],"eligible":["gather"],"move_penalty":"stone"}
]},
{"when":{"turn_equals":{"amount":6}},"then":[
{"action":"build","unit_ids":[1],"building_id":2,"max":2,"targets":"build_sites","from":"home","where":{"owner":"unowned"},"score":"var(target_enemy_distance)+var(target_region_progress)","min_score":"gold"},
{"action":"move","unit_ids":[99],"targets":"coordinate","coordinate":{"x":"wood","y":"food"}}
]},
{"when":{"always":null},"then":[{"action":"produce","prioritize":true}]}
]}`

const combatScenario = `{"rules":[{"when":{"always":null},"then":[
{"action":"attack","unit_ids":[2],"targets":"enemy_units","eligible":["gather","build","repair","renew","attack_unit"],"resume":true,"score":"var(target_health)/var(target_max_health)","min_score":"gold","spread":{"capacity":"stone","unit_order":"var(unit_stamina)","queue":1,"overflow":"none"}},
{"action":"building_attack"},
{"action":"move","unit_ids":[99],"targets":"coordinate","coordinate":{"x":"var(target_health)+var(unit_stamina)+var(target_id)","y":"var(target_assigned)+var(assign_round)"}}
]}]}`

const spatialScenario = `{"rules":[{"when":{"always":null},"then":[
{"action":"select_space","options":[{"targets":"enemy_units","score":"var(target_health)"},{"targets":"enemy_buildings","score":"var(target_health)"}],"var":"chosen","var_score":"quality","persist":true},
{"action":"move","unit_ids":[99],"unit_when":{"population_at_least":{"amount":1,"unit_ids":[2],"within":0,"from":"unit"}},"targets":"coordinate","coordinate":{"x":"var(chosen_x)","y":"var(chosen_y)+var(quality)"}},
{"action":"move","unit_ids":[2],"targets":"closest_spaces","from":"home","where":{"vision":0,"owner":"unowned"},"exclude_reference":true,"face":{"x":3,"y":0},"together":true}
]}]}`

const bucketScenario = `{"rules":[
{"when":{"turn_equals":{"amount":1}},"then":[
{"action":"set_bucket","bucket":"a","size":3,"task":{"action":"gather","category":"wood"}},
{"action":"fill_bucket","bucket":"a","unit_ids":[1],"eligible":["gather"]},
{"action":"set_bucket","bucket":"b","size":1,"task":{"action":"move","targets":"coordinate","coordinate":{"x":2,"y":0}}},
{"action":"drain_bucket","from":"a","to":"b","max":1},
{"action":"drain_bucket","from":["missing","a"],"to":"b","max":1,"allow_reserved":true}
]},
{"when":{"turn_equals":{"amount":2}},"then":[
{"action":"set_bucket","bucket":"a","size":1,"task":{"action":"move","targets":"coordinate","coordinate":{"x":3,"y":0}}},
{"action":"empty_bucket","bucket":"b"},{"action":"fill_bucket","bucket":"b","unit_ids":[1]}
]},
{"when":{"turn_equals":{"amount":3}},"then":[
{"action":"empty_bucket","bucket":"b"},{"action":"drain_bucket","from":"any","to":"b","max":1,"allow_reserved":true}
]},
{"when":{"always":null},"then":[{"action":"run_bucket","bucket":"b"},{"action":"run_bucket","bucket":"a","when_full":false},{"action":"move","unit_ids":[99],"targets":"coordinate","coordinate":{"x":"10*var(bucket_size_a)+var(bucket_size_b)","y":0}}]}
]}`
