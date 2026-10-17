package mapgen

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

type includeParams struct {
	Path string `json:"path"`
}

func stepInclude(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[includeParams](raw, "include")
	if err != nil {
		return err
	}
	if p.Path == "" {
		return fmt.Errorf("include: path cannot be empty")
	}
	path := p.Path
	if !strings.HasSuffix(path, ".json") {
		path += ".json"
	}
	clean := filepath.Clean(filepath.ToSlash(path))
	parts := append([]string{"maps", "scripted"}, strings.Split(clean, "/")...)
	data, err := defs.ReadConfigFile(parts...)
	if err != nil {
		return fmt.Errorf("include %q: %w", p.Path, err)
	}
	var steps []mapScriptStepJSON
	if err := json.Unmarshal(data, &steps); err != nil {
		return fmt.Errorf("include %q: %w", p.Path, err)
	}
	return runBranch(ctx, steps)
}
