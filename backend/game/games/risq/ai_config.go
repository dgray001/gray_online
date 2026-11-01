package risq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func validateAiConfig(name string) error {
	data, err := defs.ReadConfigFile("ai", name+".json")
	if err != nil {
		return fmt.Errorf("unknown ai config %q", name)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("ai config %q is not valid JSON: %v", name, err)
	}
	var diagnostics bytes.Buffer
	if _, unparsed := ai.ParseModelWithDiagnostics(raw, nil, &diagnostics).(ai.NoopModel); unparsed {
		return fmt.Errorf("ai config %q does not parse as an ai: %s", name, strings.TrimSpace(diagnostics.String()))
	}
	return nil
}
