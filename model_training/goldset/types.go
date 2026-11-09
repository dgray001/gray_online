package goldset

import "github.com/gin-gonic/gin"

type Card struct {
	Suit   uint8 `json:"suit"`
	Number uint8 `json:"number"`
	Index  int   `json:"index"`
	Action int   `json:"action"`
	Played bool  `json:"played"`
}
type Source struct {
	PresetID    string `json:"preset_id,omitempty"`
	Seed        int64  `json:"seed"`
	ActionSeed  int64  `json:"action_seed"`
	Decision    int    `json:"decision"`
	Players     int    `json:"players"`
	RoundPoints uint16 `json:"round_points"`
	TrickPoints uint16 `json:"trick_points"`
}
type Label struct {
	Acceptable []int            `json:"acceptable_actions"`
	Reference  *Model4Reference `json:"model4_reference,omitempty"`
	ItemID     string           `json:"item_id"`
	Action     int              `json:"action"`
	Notes      string           `json:"notes"`
	Confidence float64          `json:"confidence"`
	SavedAt    string           `json:"saved_at"`
}
type Item struct {
	slot             int
	SchemaVersion    int       `json:"schema_version"`
	ID               string    `json:"id"`
	CreatedAt        string    `json:"created_at"`
	Source           Source    `json:"source"`
	State            State     `json:"state"`
	Features         []float64 `json:"features"`
	Legal            []int     `json:"legal"`
	Analysis         gin.H     `json:"analysis"`
	Model4Parameters gin.H     `json:"model4_parameters,omitempty"`
	Model4Action     int       `json:"model4_action"`
}
type Entry struct {
	Item
	Label *Label `json:"label"`
}
type QueueItem struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Players  int    `json:"players"`
	HandSize uint8  `json:"hand_size"`
	Seat     int    `json:"seat"`
	Labelled bool   `json:"labelled"`
}
