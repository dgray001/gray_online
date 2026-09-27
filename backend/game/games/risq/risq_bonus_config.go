package risq

import (
	"encoding/json"
	"fmt"
)

var bonusConfigs []TechConfig

func loadBonusConfig(data []byte) {
	var entries []techConfigJSON
	if err := json.Unmarshal(data, &entries); err != nil {
		panic(fmt.Sprintf("failed to parse config/bonuses.json: %v", err))
	}
	bonusConfigs = make([]TechConfig, len(entries))
	for i, e := range entries {
		bonusConfigs[i] = parseTechConfigEntry(e, "config/bonuses.json tech_id")
	}
}
