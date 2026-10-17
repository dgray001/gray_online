package mapgen

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

type pathPriorityItem struct {
	space Space
	cost  float64
	index int
}

type pathPriorityQueue []*pathPriorityItem

func (pq pathPriorityQueue) Len() int           { return len(pq) }
func (pq pathPriorityQueue) Less(i, j int) bool { return pq[i].cost < pq[j].cost }
func (pq pathPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}
func (pq *pathPriorityQueue) Push(x any) {
	n := len(*pq)
	item := x.(*pathPriorityItem)
	item.index = n
	*pq = append(*pq, item)
}
func (pq *pathPriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

func buildPathCostFunc(ctx *mapScriptContext, rawCosts map[string]float64) func(Space) float64 {
	costMap := make(map[uint32]float64)
	cfg := ctx.config()
	for k, v := range rawCosts {
		cost := v
		if cost <= 0 {
			cost = 1.0
		}
		if tType, err := defs.ParseTerrainType(k); err == nil {
			for id, t := range cfg.Terrains {
				if t.Terrain_type == tType {
					costMap[id] = cost
				}
			}
			continue
		}
		if id, err := strconv.ParseUint(k, 10, 32); err == nil {
			costMap[uint32(id)] = cost
			continue
		}
		if val, ok := ctx.vars[k]; ok {
			costMap[uint32(val)] = cost
			continue
		}
		for id, t := range cfg.Terrains {
			if t.Display_name == k {
				costMap[id] = cost
			}
		}
	}
	return func(s Space) float64 {
		if c, ok := costMap[s.Terrain()]; ok {
			return c
		}
		return 100.0
	}
}

func findDijkstraPath(start Space, goal Space, costFn func(Space) float64) []Space {
	if start == nil || goal == nil {
		return nil
	}
	if start.Key() == goal.Key() {
		return []Space{start}
	}
	dist := map[uint]float64{start.Key(): 0}
	prev := map[uint]Space{start.Key(): nil}

	pq := &pathPriorityQueue{}
	heap.Init(pq)
	heap.Push(pq, &pathPriorityItem{space: start, cost: 0})

	for pq.Len() > 0 {
		top := heap.Pop(pq).(*pathPriorityItem)
		curr := top.space
		if curr.Key() == goal.Key() {
			break
		}
		if top.cost > dist[curr.Key()] {
			continue
		}
		for _, adj := range curr.SortedAdjacent() {
			stepCost := costFn(adj)
			newCost := dist[curr.Key()] + stepCost
			oldCost, seen := dist[adj.Key()]
			if !seen || newCost < oldCost {
				dist[adj.Key()] = newCost
				prev[adj.Key()] = curr
				heap.Push(pq, &pathPriorityItem{space: adj, cost: newCost})
			}
		}
	}

	if _, ok := prev[goal.Key()]; !ok {
		return nil
	}
	var path []Space
	for cur := goal; cur != nil; cur = prev[cur.Key()] {
		path = append(path, cur)
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}
func resolveEndpointCoord(raw json.RawMessage, ctx *mapScriptContext) (game_utils.Coordinate2D, error) {
	trimmed := strings.TrimSpace(string(raw))
	if strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"") {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return game_utils.Coordinate2D{}, err
		}
		if strings.HasPrefix(s, "player:") {
			numStr := strings.TrimPrefix(s, "player:")
			num, err := strconv.Atoi(numStr)
			if err != nil {
				return game_utils.Coordinate2D{}, fmt.Errorf("invalid player endpoint %q", s)
			}
			if len(ctx.player_starts) == 0 {
				return game_utils.Coordinate2D{}, fmt.Errorf("terrain_blob: path between players requires an earlier player_starts step")
			}
			if num < 1 || num > len(ctx.player_starts) {
				return game_utils.Coordinate2D{}, fmt.Errorf("player %d out of range (1..%d)", num, len(ctx.player_starts))
			}
			return ctx.player_starts[num-1].space.Coordinate(), nil
		}
		return game_utils.Coordinate2D{}, fmt.Errorf("unsupported endpoint string %q", s)
	}

	var obj struct {
		Player *int       `json:"player"`
		X      ScriptExpr `json:"x"`
		Y      ScriptExpr `json:"y"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return game_utils.Coordinate2D{}, err
	}
	if obj.Player != nil {
		if len(ctx.player_starts) == 0 {
			return game_utils.Coordinate2D{}, fmt.Errorf("terrain_blob: path between players requires an earlier player_starts step")
		}
		num := *obj.Player
		if num < 1 || num > len(ctx.player_starts) {
			return game_utils.Coordinate2D{}, fmt.Errorf("player %d out of range (1..%d)", num, len(ctx.player_starts))
		}
		return ctx.player_starts[num-1].space.Coordinate(), nil
	}
	x, err := obj.X.resolveInt(ctx.vars)
	if err != nil {
		return game_utils.Coordinate2D{}, fmt.Errorf("endpoint x: %w", err)
	}
	y, err := obj.Y.resolveInt(ctx.vars)
	if err != nil {
		return game_utils.Coordinate2D{}, fmt.Errorf("endpoint y: %w", err)
	}
	return game_utils.Coordinate2D{X: x, Y: y}, nil
}

type pathBetweenObjectJSON struct {
	Players *struct {
		Nearest int `json:"nearest"`
	} `json:"players"`
}

func resolveBetweenPairs(raw json.RawMessage, points []axialParamJSON, ctx *mapScriptContext) ([][2]game_utils.Coordinate2D, error) {
	trimmed := strings.TrimSpace(string(raw))
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var obj pathBetweenObjectJSON
		if err := json.Unmarshal(raw, &obj); err != nil {
			return nil, err
		}
		if obj.Players != nil {
			if len(ctx.player_starts) == 0 {
				return nil, fmt.Errorf("terrain_blob: path between players requires an earlier player_starts step")
			}
			nearest := obj.Players.Nearest
			if nearest <= 0 {
				nearest = 1
			}
			n := len(ctx.player_starts)
			type edgeKey struct{ a, b int }
			seen := make(map[edgeKey]bool)
			var pairs [][2]game_utils.Coordinate2D
			for i := 0; i < n; i++ {
				type neighbor struct {
					idx  int
					dist int
				}
				cI := ctx.player_starts[i].space.Coordinate()
				var list []neighbor
				for j := 0; j < n; j++ {
					if i == j {
						continue
					}
					cJ := ctx.player_starts[j].space.Coordinate()
					list = append(list, neighbor{idx: j, dist: int(game_utils.AxialDistance(cI, cJ))})
				}
				sort.Slice(list, func(a, b int) bool {
					if list[a].dist != list[b].dist {
						return list[a].dist < list[b].dist
					}
					return list[a].idx < list[b].idx
				})
				count := min(nearest, len(list))
				for m := 0; m < count; m++ {
					j := list[m].idx
					low, high := min(i, j), max(i, j)
					ek := edgeKey{low, high}
					if !seen[ek] {
						seen[ek] = true
						pairs = append(pairs, [2]game_utils.Coordinate2D{cI, ctx.player_starts[j].space.Coordinate()})
					}
				}
			}
			return pairs, nil
		}
		return nil, fmt.Errorf("unsupported between object structure")
	}

	if len(trimmed) > 0 && trimmed[0] == '[' {
		var rawItems []json.RawMessage
		if err := json.Unmarshal(raw, &rawItems); err != nil {
			return nil, err
		}
		if len(rawItems) < 2 {
			return nil, fmt.Errorf("between endpoints array requires at least 2 endpoints")
		}
		var pairs [][2]game_utils.Coordinate2D
		for i := 0; i+1 < len(rawItems); i += 2 {
			a, err := resolveEndpointCoord(rawItems[i], ctx)
			if err != nil {
				return nil, err
			}
			b, err := resolveEndpointCoord(rawItems[i+1], ctx)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, [2]game_utils.Coordinate2D{a, b})
		}
		return pairs, nil
	}

	if len(points) >= 2 {
		pts, err := resolvePoints(ctx, points)
		if err != nil {
			return nil, err
		}
		var pairs [][2]game_utils.Coordinate2D
		for i := 0; i+1 < len(pts); i += 2 {
			pairs = append(pairs, [2]game_utils.Coordinate2D{pts[i], pts[i+1]})
		}
		return pairs, nil
	}

	return nil, fmt.Errorf("seed_region path requires 'between' or 'placement.points'")
}

func pathSeedSpaces(ctx *mapScriptContext, rawBetween json.RawMessage, points []axialParamJSON, rawCosts map[string]float64) ([][]Space, error) {
	pairs, err := resolveBetweenPairs(rawBetween, points, ctx)
	if err != nil {
		return nil, err
	}
	costFn := buildPathCostFunc(ctx, rawCosts)
	var groups [][]Space
	for _, pair := range pairs {
		start := ctx.board.Space(pair[0])
		goal := ctx.board.Space(pair[1])
		if start == nil || goal == nil {
			continue
		}
		path := findDijkstraPath(start, goal, costFn)
		if len(path) > 0 {
			groups = append(groups, path)
		}
	}
	return groups, nil
}
