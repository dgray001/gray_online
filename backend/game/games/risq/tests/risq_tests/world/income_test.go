package world

import (
	"slices"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

// P0's soldiers hold (0,0) and (1,0); P1's holds (3,0), with extra P1 soldiers beside it; (2,0) is empty
func incomeGame(t *testing.T, extra string, extraP1Soldiers int) (*harness.Game, int, int) {
	t.Helper()
	p1 := unit(infantry, 1)
	for i := 0; i < extraP1Soldiers; i++ {
		p1 += "," + unit(infantry, 1)
	}
	spaces := spaceWith(0, 0, grass, unit(infantry, 0)) + "," + spaceWith(1, 0, grass, unit(infantry, 0)) + "," +
		spaceWith(2, 0, grass, "") + "," + spaceWith(3, 0, grass, p1)
	g := startGame(t, mapDoc(extra, spaces))
	return g, g.Human(0), g.Human(1)
}

const (
	spaceIncome = `,"space_gold_income":3`
	eastWest    = `,"regions":[{"name":"West","gold_bonus":5,"spaces":[[0,0],[1,0]]},{"name":"East","gold_bonus":7,"spaces":[[2,0],[3,0]]}]`
	split       = `,"regions":[{"name":"Split","gold_bonus":9,"spaces":[[1,0],[3,0]]}]`
)

func TestGoldIncomeAfterTheFirstTurn(t *testing.T) {
	cases := map[string]struct {
		extra  string
		p0, p1 float64
		why    string
	}{
		"default space income": {"", 4, 2, "two spaces at the default 2, and one"},
		"configured income":    {spaceIncome, 6, 3, "two spaces at 3, and one"},
		"fully held region":    {spaceIncome + eastWest, 11, 3, "P0 gets West's bonus of 5; P1 holds half of East so no bonus"},
		"region held by two":   {spaceIncome + split, 6, 3, "a region shared between players pays nobody"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			g, p0, p1 := incomeGame(t, c.extra, 0)
			if got := g.Self(p0).Resources.Gold; got != c.p0 {
				t.Errorf("P0 gold %v, want %v: %s", got, c.p0, c.why)
			}
			if got := g.Self(p1).Resources.Gold; got != c.p1 {
				t.Errorf("P1 gold %v, want %v: %s", got, c.p1, c.why)
			}
		})
	}
}

func TestSpacesReportTheirIncome(t *testing.T) {
	for extra, want := range map[string]float64{"": 2, spaceIncome: 3} {
		g, p0, _ := incomeGame(t, extra, 0)
		if got := g.State(p0).Space(0, 0).GoldIncome; got != want {
			t.Errorf("space gold_income %v with %q, want %v", got, extra, want)
		}
	}
}

func TestIncomeIsPaidEveryTurn(t *testing.T) {
	g, p0, _ := incomeGame(t, spaceIncome, 0)
	g.EndTurn()
	g.EndTurn()
	if got := g.Self(p0).Resources.Gold; got != 18 {
		t.Errorf("P0 gold %v after two more turns, want 18 (3 payments of 6)", got)
	}
}

func TestTakingTheRestOfARegionStartsPayingItsBonus(t *testing.T) {
	g, _, p1 := incomeGame(t, spaceIncome+eastWest, 1)
	before := g.Self(p1).Resources.Gold
	g.Submit(p1, harness.OrderMove([]uint64{g.Self(p1).Units[0].InternalID}, 2, 0))
	g.EndTurn()
	spaces := map[harness.Coord]int{}
	for _, u := range g.Self(p1).Units {
		spaces[u.Space]++
	}
	if spaces[harness.Coord{X: 2, Y: 0}] != 1 || spaces[harness.Coord{X: 3, Y: 0}] != 1 {
		t.Fatalf("P1 soldiers at %v, want one in each of (2,0) and (3,0)", spaces)
	}
	if got := g.Self(p1).Resources.Gold - before; got != 13 {
		t.Errorf("P1 earned %v this turn, want 13 (two spaces at 3 plus East's bonus of 7)", got)
	}
}

func regionNamed(regions []harness.Region, name string) *harness.Region {
	for i := range regions {
		if regions[i].Name == name {
			return &regions[i]
		}
	}
	return nil
}

// A region is served once any of its spaces is explored, listing only the explored ones
func TestRegionsListOnlyTheirExploredSpaces(t *testing.T) {
	g, p0, p1 := incomeGame(t, spaceIncome+eastWest, 0)
	p0Regions, p1Regions := g.State(p0).Regions, g.State(p1).Regions
	if west := regionNamed(p0Regions, "West"); west == nil || len(west.Spaces) != 2 || west.Owner != 0 {
		t.Errorf("P0's West %+v, want both spaces, owned by P0", west)
	}
	if east := regionNamed(p0Regions, "East"); east == nil || !slices.Equal(east.Spaces, []uint{uint(harness.SpaceKey(2, 0))}) {
		t.Errorf("P0's East %+v, want only the key of (2,0), the one space P0 can see", east)
	}
	if regionNamed(p1Regions, "West") != nil {
		t.Error("P1 is told about West without having explored any of it")
	}
	if east := regionNamed(p1Regions, "East"); east == nil || len(east.Spaces) != 2 {
		t.Errorf("P1's East %+v, want both its spaces", east)
	}
}

// By design a region's owner is reported live to anyone who has explored any of its spaces
func TestRegionOwnerIsReportedLiveOnceAnySpaceIsExplored(t *testing.T) {
	g, p0, p1 := incomeGame(t, spaceIncome+eastWest, 1)
	g.Submit(p1, harness.OrderMove([]uint64{g.Self(p1).Units[0].InternalID}, 2, 0))
	g.EndTurn()
	east := regionNamed(g.State(p0).Regions, "East")
	if east == nil || !slices.Equal(east.Spaces, []uint{uint(harness.SpaceKey(2, 0))}) || east.Owner != g.PlayerID(p1) {
		t.Errorf("P0's East %+v, want only the key of (2,0), with P1 already named as owner", east)
	}
}
