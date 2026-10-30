package risq

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/rand"
	"regexp"
	"sort"
	"strings"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/internal/mapgen"
	"github.com/gin-gonic/gin"
)

func PreviewCustomMap(doc []byte) (gin.H, error) {
	var header struct {
		Players int `json:"players"`
	}
	if err := json.Unmarshal(doc, &header); err != nil {
		return nil, err
	}
	if header.Players < 1 || header.Players > 12 {
		return nil, errors.New("players must be 1 to 12")
	}
	ai_players := make([]any, header.Players)
	for i := range ai_players {
		ai_players[i] = map[string]any{"nickname": "slot-" + string(rune('a'+i))}
	}
	base := game.CreateBaseGame(0, game.GameType_RISQ, map[string]any{"ai_players": ai_players, "seed": float64(1)}, nil, nil)
	generate := func(board mapgen.Board, _ *rand.Rand, num_players int) error {
		return mapgen.GenerateCustomDocument(board, num_players, doc)
	}
	config_dir, err := mapgen.CustomDocumentConfigDir(doc)
	if err != nil {
		return nil, err
	}
	config, err := gameConfig(config_dir)
	if err != nil {
		return nil, err
	}
	r, err := createGame(base, make(chan game.PlayerAction, 16), 1, config, generate)
	if err != nil {
		return nil, err
	}
	defer r.StopAi()
	for _, space := range r.allSpaces() {
		space.visibility[-1] = defs.VisibilitySpy
	}
	return r.toFrontendFor(-1, 0, true, true), nil
}

var customMapNamePattern = regexp.MustCompile(`^[a-z0-9_]{1,40}$`)

func ListCustomMaps() ([]string, error) {
	entries, err := defs.ListConfigDir("maps", "custom")
	names := []string{}
	for _, e := range entries {
		if name, ok := strings.CutSuffix(e.Name(), ".json"); ok {
			names = append(names, name)
		}
	}
	return names, err
}

func ReadCustomMap(name string) ([]byte, error) {
	if !customMapNamePattern.MatchString(name) {
		return nil, errors.New("map names are lowercase letters, digits and underscores")
	}
	return defs.ReadConfigFile("maps", "custom", name+".json")
}

func SaveCustomMap(name string, doc []byte) error {
	if !customMapNamePattern.MatchString(name) {
		return errors.New("map names are lowercase letters, digits and underscores")
	}
	if _, err := PreviewCustomMap(doc); err != nil {
		return err
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, doc, "", "  "); err != nil {
		return err
	}
	return defs.WriteConfigFile(append(pretty.Bytes(), '\n'), "maps", "custom", name+".json")
}

func AllTerrainConfigsToFrontend(game_config *defs.Config) []gin.H {
	terrains := make([]gin.H, 0, len(game_config.Terrains))
	for id, config := range game_config.Terrains {
		terrains = append(terrains, gin.H{"terrain_id": id, "display_name": config.Display_name, "terrain_type": config.Terrain_type})
	}
	sort.Slice(terrains, func(i, j int) bool { return terrains[i]["terrain_id"].(uint32) < terrains[j]["terrain_id"].(uint32) })
	return terrains
}

func sortedEntities[V any](configs map[uint32]V, display_name func(V) string) []gin.H {
	entities := make([]gin.H, 0, len(configs))
	for id, config := range configs {
		entities = append(entities, gin.H{"id": id, "display_name": display_name(config)})
	}
	sort.Slice(entities, func(i, j int) bool { return entities[i]["id"].(uint32) < entities[j]["id"].(uint32) })
	return entities
}

func AllEntityConfigsToFrontend(config *defs.Config) gin.H {
	return gin.H{
		"buildings": sortedEntities(config.Buildings, func(c defs.BuildingConfig) string { return c.Display_name }),
		"units":     sortedEntities(config.Units, func(c defs.UnitConfig) string { return c.Display_name }),
		"resources": sortedEntities(config.Resources, func(c defs.ResourceConfig) string { return c.Display_name }),
	}
}

// The tables a map's config folder resolves to, "" being the default tables
func ConfigFromDir(config_dir string) (*defs.Config, error) {
	return gameConfig(config_dir)
}

func DeleteCustomMap(name string) error {
	if !customMapNamePattern.MatchString(name) {
		return errors.New("map names are lowercase letters, digits and underscores")
	}
	return defs.RemoveConfigFile("maps", "custom", name+".json")
}
