package risq

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed config/bonuses.json
var bonusesConfigJSON []byte

var bonusConfigs []TechConfig

func init() {
	var entries []techConfigJSON
	if err := json.Unmarshal(bonusesConfigJSON, &entries); err != nil {
		panic(fmt.Sprintf("failed to parse config/bonuses.json: %v", err))
	}
	bonusConfigs = make([]TechConfig, len(entries))
	for i, e := range entries {
		bonusConfigs[i] = parseTechConfigEntry(e, "config/bonuses.json tech_id")
	}
}
