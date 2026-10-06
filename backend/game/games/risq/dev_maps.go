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
	base := game.CreateBaseGame(0, game.GameType_RISQ, map[string]any{"ai_players": ai_players, "seed": float64(1)})
	generate := func(board mapgen.Board, _ *rand.Rand, num_players int) error {
		return mapgen.GenerateCustomDocument(board, num_players, doc)
	}
	r, err := createGame(base, make(chan game.PlayerAction, 16), 1, generate)
	if err != nil {
		return nil, err
	}
	defer r.StopAi()
	for _, space := range r.allSpaces() {
		space.visibility[-1] = defs.VisibilitySpy
	}
	return r.toFrontendFor(-1, 0, true), nil
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

func AllTerrainConfigsToFrontend() []gin.H {
	terrains := make([]gin.H, 0, len(defs.TerrainConfigs))
	for id, config := range defs.TerrainConfigs {
		terrains = append(terrains, gin.H{"terrain_id": id, "display_name": config.Display_name, "terrain_type": config.Terrain_type})
	}
	sort.Slice(terrains, func(i, j int) bool { return terrains[i]["terrain_id"].(uint32) < terrains[j]["terrain_id"].(uint32) })
	return terrains
}

func DeleteCustomMap(name string) error {
	if !customMapNamePattern.MatchString(name) {
		return errors.New("map names are lowercase letters, digits and underscores")
	}
	return defs.RemoveConfigFile("maps", "custom", name+".json")
}
