package risq

import (
	"errors"
	"math/rand"
	"strconv"
	"time"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/util"
)

// palette a player's color defaults to when the lobby doesn't request one; order = default assignment order
var risqPlayerColors = []string{
	"90, 90, 250",   // blue
	"250, 60, 60",   // red
	"60, 220, 60",   // green
	"250, 230, 50",  // yellow
	"60, 230, 230",  // cyan
	"160, 70, 230",  // purple
	"250, 150, 30",  // orange
	"250, 120, 190", // pink
	"170, 170, 170", // gray
	"150, 90, 40",   // brown
	"50, 60, 160",   // navy
	"170, 250, 80",  // lime
}

// keyed by client_id string since that's how game_specific_settings arrives off the wire (raw JSON)
func requestedRisqPlayerColors(g *game.GameBase) map[uint64]string {
	requested := make(map[uint64]string)
	raw, ok := g.GameSpecificSettings["player_colors"].(map[string]interface{})
	if !ok {
		return requested
	}
	for client_id_str, color_raw := range raw {
		color, color_ok := color_raw.(string)
		if !color_ok {
			continue
		}
		client_id, err := strconv.ParseUint(client_id_str, 10, 64)
		if err != nil {
			continue
		}
		requested[client_id] = color
	}
	return requested
}

func nextAvailableRisqColor(used map[string]bool) string {
	for _, color := range risqPlayerColors {
		if !used[color] {
			return color
		}
	}
	return ""
}

func CreateGame(g *game.GameBase, action_channel chan game.PlayerAction) (*GameRisq, error) {
	seed, ok := g.GameSpecificSettings["seed"].(float64)
	if !ok {
		seed = float64(time.Now().UnixNano())
	}
	risq := GameRisq{
		game:                      g,
		players:                   []*RisqPlayer{},
		units:                     make(map[uint64]*RisqUnit),
		buildings:                 make(map[uint64]*RisqBuilding),
		population_limit:          100,
		next_resource_internal_id: 0,
		next_building_internal_id: 0,
		next_unit_internal_id:     0,
		next_order_internal_id:    0,
		turn_number:               0,
		rng:                       rand.New(rand.NewSource(int64(seed))),
	}
	requested_colors := requestedRisqPlayerColors(g)
	used_colors := make(map[string]bool)
	for _, color := range requested_colors {
		used_colors[color] = true
	}
	var player_id = 0
	for _, player := range g.Players {
		player.Player_id = player_id
		color, requested := requested_colors[player.GetClientId()]
		if !requested {
			color = nextAvailableRisqColor(used_colors)
			used_colors[color] = true
		}
		player_rng := rand.New(rand.NewSource(int64(seed) + int64(player_id) + 1))
		risq.players = append(risq.players, createRisqPlayer(player, risq.population_limit, color, player_rng))
		player_id++
	}
	ai_players, ai_players_ok := g.GameSpecificSettings["ai_players"].([]any)
	ai_risq_players := make([]*RisqPlayer, 0)
	if ai_players_ok {
		for _, ai_player := range ai_players {
			ai, ai_ok := ai_player.(map[string]any)
			if !ai_ok {
				continue
			}
			nickname, nickname_ok := ai["nickname"].(string)
			if !nickname_ok {
				continue
			}
			config, config_ok := ai["config"].(string)
			if !config_ok {
				config = "default"
			}
			player := game.CreateAiPlayer(nickname, g)
			player.Player_id = player_id
			color := nextAvailableRisqColor(used_colors)
			used_colors[color] = true
			player_rng := rand.New(rand.NewSource(int64(seed) + int64(player_id) + 1))
			risq_player := createRisqPlayer(player, risq.population_limit, color, player_rng)
			risq_player.createAiModel(config)
			ai_risq_players = append(ai_risq_players, risq_player)
			risq.players = append(risq.players, risq_player)
			player_id++
		}
	}
	if len(risq.players) < 2 {
		return nil, errors.New("Need at least two players to play risq")
	} else if len(risq.players) > 12 {
		return nil, errors.New("Can have max of twelve players playing risq")
	}
	map_name, ok := g.GameSpecificSettings["map"].(string)
	if !ok || map_name == "" {
		map_name = "ring"
	}
	steps, err := loadMapScript(map_name)
	if err != nil {
		return nil, err
	}
	ctx := &mapScriptContext{
		risq:        &risq,
		rng:         risq.rng,
		num_players: len(risq.players),
		regions:     make(map[string]map[uint]bool),
		vars:        newMapScriptVars(len(risq.players), 0, 0),
	}
	if err := runMapScript(ctx, steps); err != nil {
		return nil, err
	}
	if len(ctx.player_starts) != len(risq.players) {
		return nil, errors.New("map script did not place all player starts")
	}
	risq.logBoard()
	for _, ai_player := range ai_risq_players {
		go runAi(ai_player, &risq, action_channel)
	}
	return &risq, nil
}

func (r *GameRisq) logBoard() {
	for _, space := range r.allSpaces() {
		util.DebugLog.Printf("board: space (%d,%d) terrain_id=%d", space.coordinate.X, space.coordinate.Y, space.terrain_id)
		for _, zrow := range space.zones {
			for _, zone := range zrow {
				if zone.resource != nil {
					util.DebugLog.Printf("board:   zone (%d,%d) key=%d resource_id=%d category=%d amount=%.0f",
						zone.coordinate.X, zone.coordinate.Y, zone.coordinate_key,
						zone.resource.resource_id, zone.resource.category(), zone.resource.resources_left)
				}
				if zone.building != nil {
					util.DebugLog.Printf("board:   zone (%d,%d) key=%d building_id=%d player=%d",
						zone.coordinate.X, zone.coordinate.Y, zone.coordinate_key,
						zone.building.building_id, zone.building.player_id)
				}
			}
		}
	}
}

var uniformFoodIds = []uint32{1, 2}
var uniformWoodIds = []uint32{11, 12, 13, 14, 15, 16}
var uniformStoneIds = []uint32{21}
