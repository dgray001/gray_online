package goldset

import (
	"encoding/json"
)

type Model4Reference struct {
	Model         string          `json:"model"`
	CapturedAt    string          `json:"captured_at"`
	SampledAction int             `json:"sampled_action"`
	Analysis      json.RawMessage `json:"analysis"`
	State         json.RawMessage `json:"state"`
	Parameters    json.RawMessage `json:"parameters"`
	SHA256        string          `json:"sha256"`
	Origin        string          `json:"origin"`
}
