package ai

import "fmt"

const allBuckets = "all"

type selectSink struct {
	set, size_var, bucket string
	mode                  string
	persist               bool
}

func parseSink(raw any, pick *selectorSpec) (*selectSink, error) {
	if pick != nil && (pick.kind == "orders" || pick.kind == "assignments") {
		return nil, fmt.Errorf("%s cannot be stored in sets or buckets", pick.kind)
	}
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("must be an object")
	}
	if err := checkKeys(obj, "set", "persist", "var", "bucket", "mode"); err != nil {
		return nil, err
	}
	if err := checkOptionTypes[string](obj, "set", "var", "bucket", "mode"); err != nil {
		return nil, err
	}
	if err := checkOptionTypes[bool](obj, "persist"); err != nil {
		return nil, err
	}
	s := &selectSink{mode: "replace"}
	s.set, _ = obj["set"].(string)
	s.bucket, _ = obj["bucket"].(string)
	s.size_var, _ = obj["var"].(string)
	s.persist, _ = obj["persist"].(bool)
	if (s.set == "") == (s.bucket == "") || (s.set != "" && !validVarName(s.set)) || (s.bucket != "" && !validVarName(s.bucket)) {
		return nil, fmt.Errorf("needs exactly one valid \"set\" or \"bucket\" name")
	}
	if s.bucket != "" && pick != nil && pick.kind != "units" {
		return nil, fmt.Errorf("only units can go into a bucket")
	}
	if _, present := obj["persist"]; present && s.bucket != "" {
		return nil, fmt.Errorf("\"persist\" only applies to set sinks")
	}
	if raw, present := obj["mode"]; present {
		if s.bucket == "" {
			return nil, fmt.Errorf("\"mode\" only applies to bucket sinks")
		}
		if s.mode, _ = raw.(string); s.mode != "replace" && s.mode != "add" && s.mode != "remove" {
			return nil, fmt.Errorf("bucket mode must be \"replace\", \"add\" or \"remove\"")
		}
	}
	if s.bucket == allBuckets && s.mode != "remove" {
		return nil, fmt.Errorf("the bucket name %q is reserved for mode \"remove\", which frees units from every bucket", allBuckets)
	}
	if _, present := obj["var"]; present {
		if !validVarName(s.size_var) {
			return nil, fmt.Errorf("\"var\" must be letters, digits and underscores")
		}
		noteVarSet(s.size_var)
	}
	return s, nil
}

func (s *selectSink) store(kind string, items []selItem, internals *Internals) {
	if s.size_var != "" {
		if internals.turn_vars == nil {
			internals.turn_vars = make(map[string]float64)
		}
		internals.turn_vars[s.size_var] = float64(len(items))
	}
	if s.bucket == allBuckets {
		for _, item := range items {
			if key, ok := keyOf(item); ok {
				for _, b := range internals.Buckets {
					delete(b.Members, key.id)
				}
			}
		}
		return
	}
	if s.bucket != "" {
		b := internals.bucket(s.bucket)
		if s.mode == "replace" {
			b.Members = make(map[uint64]bool, len(items))
		}
		for _, item := range items {
			if key, ok := keyOf(item); ok {
				if s.mode == "remove" {
					delete(b.Members, key.id)
				} else {
					b.Members[key.id] = true
				}
			}
		}
		if s.mode == "replace" {
			b.Desired = max(b.Desired, len(b.Members))
		}
		return
	}
	set := &entitySet{kind: kind, keys: make([]itemKey, 0, len(items))}
	for _, item := range items {
		if key, ok := keyOf(item); ok {
			set.keys = append(set.keys, key)
		}
	}
	internals.storeSet(s.set, set, s.persist)
}
