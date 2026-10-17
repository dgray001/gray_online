package risq_mapgen_tests

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"testing"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func renderBoardToPNG(board *fakeboard.Board, filename string) error {
	const (
		hexR = 24.0
		pad  = 40.0
	)
	hexW := math.Sqrt(3) * hexR
	hexH := 1.5 * hexR

	infos := board.Inspect()
	minX, maxX := math.MaxFloat64, -math.MaxFloat64
	minY, maxY := math.MaxFloat64, -math.MaxFloat64

	centers := make(map[game_utils.Coordinate2D][2]float64)
	for _, info := range infos {
		q, r := float64(info.Coord.X), float64(info.Coord.Y)
		cx := hexW * (q + 0.5*r)
		cy := hexH * r
		centers[info.Coord] = [2]float64{cx, cy}
		minX = min(minX, cx-hexW/2)
		maxX = max(maxX, cx+hexW/2)
		minY = min(minY, cy-hexR)
		maxY = max(maxY, cy+hexR)
	}

	w := int(math.Ceil(maxX - minX + 2*pad))
	h := int(math.Ceil(maxY - minY + 2*pad))
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	bg := color.RGBA{18, 22, 28, 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, bg)
		}
	}

	regionPalette := []color.RGBA{
		{230, 195, 75, 255}, {215, 80, 75, 255}, {225, 140, 50, 255},
		{115, 185, 65, 255}, {50, 185, 175, 255}, {100, 130, 230, 255},
		{185, 90, 205, 255},
	}
	coordToRegion := make(map[game_utils.Coordinate2D]int)
	for rIdx, r := range board.Regions {
		for _, s := range board.Spaces() {
			if r.Keys[s.Key()] {
				coordToRegion[s.Coordinate()] = rIdx
			}
		}
	}
	playerStarts := make(map[game_utils.Coordinate2D]int)
	for _, info := range infos {
		for _, z := range info.Zones {
			if z.Building.ID == 1 {
				playerStarts[info.Coord] = z.Building.Player
			}
		}
	}
	borderCol := color.RGBA{30, 40, 50, 180}
	for _, info := range infos {
		c := centers[info.Coord]
		cx := c[0] - minX + pad
		cy := c[1] - minY + pad

		col := color.RGBA{86, 150, 75, 255}
		if info.Terrain >= 301 {
			col = color.RGBA{22, 60, 115, 255}
		} else if info.Terrain >= 251 {
			col = color.RGBA{55, 145, 215, 255}
		} else if rIdx, ok := coordToRegion[info.Coord]; ok && rIdx < len(regionPalette) {
			col = regionPalette[rIdx]
		}

		for dy := -int(hexR); dy <= int(hexR); dy++ {
			for dx := -int(hexW / 2); dx <= int(hexW/2); dx++ {
				py := float64(dy)
				px := float64(dx)
				if math.Abs(px) <= hexW/2 && math.Abs(py) <= hexR && (hexW/2)*math.Abs(py)+(hexR/2)*math.Abs(px) <= (hexW/2)*hexR {
					ix := int(math.Round(cx + px))
					iy := int(math.Round(cy + py))
					if ix >= 0 && ix < w && iy >= 0 && iy < h {
						if math.Abs(px) > hexW/2-1.5 || math.Abs(py) > hexR-1.5 || (hexW/2)*math.Abs(py)+(hexR/2)*math.Abs(px) > (hexW/2)*hexR-2.0 {
							img.Set(ix, iy, borderCol)
						} else {
							img.Set(ix, iy, col)
						}
					}
				}
			}
		}
	}
	for coord := range playerStarts {
		c := centers[coord]
		cx := int(math.Round(c[0] - minX + pad))
		cy := int(math.Round(c[1] - minY + pad))
		r := 8
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				distSq := dx*dx + dy*dy
				if distSq <= r*r {
					dotCol := color.RGBA{255, 255, 255, 255}
					if distSq <= (r-3)*(r-3) {
						dotCol = color.RGBA{15, 15, 20, 255}
					}
					img.Set(cx+dx, cy+dy, dotCol)
				}
			}
		}
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func TestRenderAsteriskMap(t *testing.T) {
	if err := defs.LoadConfig("../../config"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = defs.LoadConfig(testConfig) }()
	board, err := fakeboard.GenerateWithSize("script:asterisk", 9, defs.MapSize_LARGE, 1)
	if err != nil {
		t.Fatal(err)
	}
	outFile := "/home/yodan/.gemini/antigravity-cli/brain/60c3a88b-aead-409b-aeee-9c675f1bb1c9/asterisk_preview.png"
	if err := renderBoardToPNG(board, outFile); err != nil {
		t.Fatal(err)
	}
}
