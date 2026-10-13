package mapgen

import (
	"encoding/json"
	"fmt"
)

type ifParams struct {
	// the branch runs when this evaluates above 0, e.g. max(0, 1 - biome * biome) holds only for biome 0
	Condition ScriptExpr          `json:"condition"`
	Then      []mapScriptStepJSON `json:"then"`
	Else      []mapScriptStepJSON `json:"else"`
}

func stepIf(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[ifParams](raw, "if")
	if err != nil {
		return err
	}
	value, err := p.Condition.resolve(ctx.vars)
	if err != nil {
		return err
	}
	branch := p.Else
	if value > 0 {
		branch = p.Then
	}
	return runBranch(ctx, branch)
}

func runBranch(ctx *mapScriptContext, steps []mapScriptStepJSON) error {
	for _, s := range steps {
		fn, ok := mapStepRegistry[s.Step]
		if !ok {
			return fmt.Errorf("unknown map script step %q", s.Step)
		}
		if err := fn(ctx, s.Params); err != nil {
			return fmt.Errorf("map script step %q: %w", s.Step, err)
		}
	}
	return nil
}
