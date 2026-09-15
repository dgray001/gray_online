package risq

import (
	"errors"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
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
			go runAi(risq_player, &risq, action_channel)
			risq.players = append(risq.players, risq_player)
			player_id++
		}
	}
	if len(risq.players) < 2 {
		return nil, errors.New("Need at least two players to play risq")
	} else if len(risq.players) > 12 {
		return nil, errors.New("Can have max of twelve players playing risq")
	}
	starting_distance := 0
	switch len(risq.players) {
	case 6:
		risq.board_size = 6
		starting_distance = util.RandomIntFrom(risq.rng, 4, 5)
	case 5:
		risq.board_size = 6
		starting_distance = util.RandomIntFrom(risq.rng, 4, 5)
	case 4:
		risq.board_size = 5
		starting_distance = util.RandomIntFrom(risq.rng, 3, 4)
	case 3:
		risq.board_size = 4
		starting_distance = util.RandomIntFrom(risq.rng, 3, 3)
	default:
		risq.board_size = 4
		starting_distance = util.RandomIntFrom(risq.rng, 4, 4)
	}
	if override, ok := g.GameSpecificSettings["board_size"].(float64); ok && override >= 2 {
		risq.board_size = uint16(override)
	}
	if override, ok := g.GameSpecificSettings["starting_distance"].(float64); ok && override >= 0 {
		starting_distance = int(override)
	}
	if starting_distance > int(risq.board_size) {
		starting_distance = int(risq.board_size)
	}
	starting_units := map[uint32]int{1: 3, 11: 1}
	if raw, ok := g.GameSpecificSettings["starting_units"].(map[string]any); ok {
		starting_units = make(map[uint32]int, len(raw))
		for id_str, count_raw := range raw {
			id, err := strconv.ParseUint(id_str, 10, 32)
			count, count_ok := count_raw.(float64)
			if err != nil || !count_ok {
				continue
			}
			starting_units[uint32(id)] = int(count)
		}
	}
	risq.spaces = make([][]*RisqSpace, 2*int(risq.board_size)+1)
	for j := range risq.spaces {
		r := j - int(risq.board_size)
		l := 2*int(risq.board_size) + 1 - util.AbsInt(r)
		risq.spaces[j] = make([]*RisqSpace, l)
		for i := range risq.spaces[j] {
			q := max(-int(risq.board_size), -(int(risq.board_size)+r)) + i
			risq.spaces[j][i] = createRisqSpace(q, r, defaultTerrainId)
		}
	}
	for _, row := range risq.spaces {
		for _, space := range row {
			for _, v := range game_utils.AxialDirectionVectors() {
				adjacent := risq.getSpace(space.coordinate.Add(&v))
				if adjacent != nil {
					space.setAdjacentSpace(adjacent, &v)
				}
			}
		}
	}
	starting_location := util.RandomIntFrom(risq.rng, 0, 5)
	axial_unit_vectors := game_utils.AxialDirectionVectors()
	direction_offsets := map[int][]int{
		1: {0},
		2: {0, 3},
		3: {0, 2, 4},
		4: {0, 1, 3, 4},
		5: {0, 1, 2, 3, 4},
		6: {0, 1, 2, 3, 4, 5},
	}
	offsets, ok := direction_offsets[len(risq.players)]
	if !ok {
		return nil, errors.New("unknown number of players")
	}
	placed := make(map[uint]bool)
	for i, offset := range offsets {
		space := risq.getSpace(axial_unit_vectors[(starting_location+offset)%6].Multiply(starting_distance))
		if space == nil {
			return nil, errors.New("starting space is nil")
		}
		risq.createPlayerStart(risq.players[i], space, starting_units)
		placed[space.coordinate_key] = true
		for _, v := range axial_unit_vectors {
			ring_space := risq.getSpace(space.coordinate.Add(&v))
			if ring_space == nil || placed[ring_space.coordinate_key] {
				continue
			}
			risq.placeFixedResourceSet(ring_space)
			placed[ring_space.coordinate_key] = true
		}
	}
	for _, row := range risq.spaces {
		for _, space := range row {
			if !placed[space.coordinate_key] {
				risq.placeUniformResources(space)
			}
		}
	}
	risq.logBoard()
	return &risq, nil
}

func (r *GameRisq) logBoard() {
	for _, row := range r.spaces {
		for _, space := range row {
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
}

func (r *GameRisq) createPlayerStart(p *RisqPlayer, s *RisqSpace, starting_units map[uint32]int) {
	village_center := createRisqBuilding(r.nextBuildingInternalId(), 1, p.player.Player_id)
	s.setBuilding(&game_utils.Coordinate2D{X: 0, Y: 0}, village_center)
	p.buildings[village_center.internal_id] = village_center
	r.buildings[village_center.internal_id] = village_center
	unit_ids := make([]uint32, 0, len(starting_units))
	for unit_id := range starting_units {
		unit_ids = append(unit_ids, unit_id)
	}
	sort.Slice(unit_ids, func(i, j int) bool { return unit_ids[i] < unit_ids[j] })
	for _, unit_id := range unit_ids {
		for range starting_units[unit_id] {
			unit := createRisqUnit(r.nextUnitInternalId(), unit_id, p)
			s.setUnit(&game_utils.Coordinate2D{X: 0, Y: 0}, unit)
			p.units[unit.internal_id] = unit
			r.units[unit.internal_id] = unit
		}
	}
	r.placeFixedResourceSet(s)
}

// Fixed 5-node set (2 food, 2 wood, 1 stone) used for player starts and the ring around them
func (r *GameRisq) placeFixedResourceSet(s *RisqSpace) {
	zones := s.getZonesAsRandomArray(false, r.rng)
	resource_ids := []uint32{1, 2, 11, 14, 21} // forage, deer, cedar, oak, stonemine
	for i, resource_id := range resource_ids {
		s.setResource(&zones[i].coordinate, createRisqResource(r.nextResourceInternalId(), resource_id))
	}
}

// Probability an outer zone gets a resource node in the general (non-fixed) map
const uniformResourceChance = 0.3

var uniformFoodIds = []uint32{1, 2}
var uniformWoodIds = []uint32{11, 12, 13, 14, 15, 16}
var uniformStoneIds = []uint32{21}

func (r *GameRisq) placeUniformResources(s *RisqSpace) {
	for _, zone := range s.getZonesAsRandomArray(false, r.rng) {
		if r.rng.Float64() >= uniformResourceChance {
			continue
		}
		var ids []uint32
		switch roll := r.rng.Float64(); {
		case roll < 0.4:
			ids = uniformFoodIds
		case roll < 0.8:
			ids = uniformWoodIds
		default:
			ids = uniformStoneIds
		}
		resource_id := ids[r.rng.Intn(len(ids))]
		s.setResource(&zone.coordinate, createRisqResource(r.nextResourceInternalId(), resource_id))
	}
}
