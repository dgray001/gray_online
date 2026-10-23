package ai

import "fmt"

func (a *selectAction) parseControl(raw map[string]any) error {
	if a.actors == nil || (a.actors.kind != "units" && a.actors.kind != "buildings") {
		return fmt.Errorf("select: controls need explicit unit or building actors")
	}
	if raw["pick"] != nil || raw["assign"] != nil || raw["mode"] != nil {
		return fmt.Errorf("select: controls take no pick, assign or mode")
	}
	for _, v := range a.verbs {
		if !v.control() {
			return fmt.Errorf("select: controls cannot be mixed with order verbs")
		}
	}
	return nil
}

func parseControlVerb(v verb, options map[string]any, actor_kind string) (verb, error) {
	if actor_kind != "units" && actor_kind != "buildings" {
		return v, fmt.Errorf("%s needs unit or building actors", v.name)
	}
	if v.name == "stop" {
		var err error
		v.stop, err = parseStopSpec(options, actor_kind)
		return v, err
	}
	if err := checkBehaviorOptions(options, actor_kind); err != nil {
		return v, err
	}
	return parseBehaviorVerb(v, options, actor_kind)
}

func (c *issueContext) control(v verb, actor loopItem) []Order {
	if v.name == "stop" {
		return c.stopOrders(v.stop, actor)
	}
	c.setBehavior(v, actor)
	return nil
}
