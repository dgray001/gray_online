package goldset

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

func freezeReference(item Item, origin string) (*Model4Reference, error) {
	analysis, err := json.Marshal(item.Analysis)
	if err != nil {
		return nil, err
	}
	state, err := json.Marshal(item.State)
	if err != nil {
		return nil, err
	}
	parameters, err := json.Marshal(item.Model4Parameters)
	if err != nil {
		return nil, err
	}
	reference := &Model4Reference{Model: "Model4", CapturedAt: item.CreatedAt, SampledAction: item.Model4Action, Analysis: analysis, State: state, Parameters: parameters}
	data, err := json.Marshal(reference)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(data)
	reference.SHA256, reference.Origin = fmt.Sprintf("%x", hash[:]), origin
	return reference, nil
}
