package neural

import (
	"encoding/json"
	"os"
)

func (n *Network) Save(path string) error {
	data, err := json.Marshal(n)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func Load(path string) (*Network, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var network Network
	if err := json.Unmarshal(data, &network); err != nil {
		return nil, err
	}
	return &network, network.validate()
}
