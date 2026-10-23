package combat

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"github.com/gin-gonic/gin"
)

const militia = 12

var attackBackStances = map[string]defs.UnitStance{
	"passive": defs.UnitStance_PASSIVE, "stand-ground": defs.UnitStance_STAND_GROUND,
	"defensive": defs.UnitStance_DEFENSIVE, "aggressive": defs.UnitStance_AGGRESSIVE,
}

func attackBackGame(t *testing.T, tweak func(), spaces ...string) *harness.Game {
	t.Helper()
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"attack-back": fmt.Sprintf(meetingBoard, strings.Join(spaces, ","))})
	tweak()
	return harness.NewGame(t, "custom:attack-back", 1, 2)
}

func setVictimBehavior(g *harness.Game, stance defs.UnitStance, interrupt bool) {
	g.Action(g.Human(0), "set-unit-behavior", gin.H{"internal_ids": []uint64{soleUnit(g, 0).InternalID}, "stance": uint8(stance), "attack_back": true, "interrupt_current": interrupt})
}

// The enemy's units in id order, so index 0 is the first one placed
func foesByID(g *harness.Game) []harness.Unit {
	foes := slices.Clone(g.Self(g.Human(1)).Units)
	slices.SortFunc(foes, func(a, b harness.Unit) int { return int(a.InternalID) - int(b.InternalID) })
	return foes
}

func setTurnStamina(id uint32, stamina int) {
	config := defs.UnitConfigs[id]
	config.Turn_stamina = stamina
	defs.UnitConfigs[id] = config
}

// A victim that cannot afford its order's step stays put, so the attacker keeps hitting it in melee range.
// A passive victim with interrupt off answers only when idle; everyone else also leaves non-attack orders.
func TestAttackBackInterruptsNonAttackOrders(t *testing.T) {
	for name, stance := range attackBackStances {
		for _, interrupt := range []bool{false, true} {
			for _, ordered := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s-interrupt-%v-ordered-%v", name, interrupt, ordered), func(t *testing.T) {
					g := attackBackGame(t, func() { setTurnStamina(blunt, 5) }, spaceOf(0, 0, flat, placed{blunt, 0, 0, 0}, placed{heavy, 1, 0, 0}), spaceOf(1, 0, flat))
					setVictimBehavior(g, stance, interrupt)
					victim, attacker := soleUnit(g, 0), soleUnit(g, 1)
					if ordered {
						g.Submit(g.Human(0), harness.OrderMove([]uint64{victim.InternalID}, 1, 0))
					} else {
						g.Submit(g.Human(0))
					}
					g.Submit(g.Human(1), attackOrder(attacker, victim))
					if want := !ordered || interrupt || stance != defs.UnitStance_PASSIVE; hurt(soleUnit(g, 1)) != want {
						t.Fatalf("victim attacked back=%v, want %v", !want, want)
					}
				})
			}
		}
	}
}

// An explicit attack order yields to attack-back only when interrupt is on
func TestAttackBackVersusAssignedAttack(t *testing.T) {
	for name, stance := range attackBackStances {
		for _, interrupt := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s-interrupt-%v", name, interrupt), func(t *testing.T) {
				g := attackBackGame(t, func() {}, spaceOf(0, 0, flat, placed{blunt, 0, 0, 0}, placed{heavy, 1, 0, 0}, placed{heavy, 1, 0, 0}))
				foes := foesByID(g)
				passive(g, g.Human(1), foes[1].InternalID)
				setVictimBehavior(g, stance, interrupt)
				g.Submit(g.Human(0), attackOrder(soleUnit(g, 0), foes[1]))
				g.Submit(g.Human(1), attackOrder(foes[0], soleUnit(g, 0)))
				if attackerHurt := hurt(foesByID(g)[0]); attackerHurt != interrupt {
					t.Fatalf("attacker hurt=%v, want %v", attackerHurt, interrupt)
				}
			})
		}
	}
}

// A stance's own auto-attack on a lower-priority target gives way to a better attacker only when interrupt is on
func TestAttackBackVersusStanceAutoAttack(t *testing.T) {
	for _, interrupt := range []bool{false, true} {
		t.Run(fmt.Sprintf("interrupt-%v", interrupt), func(t *testing.T) {
			tweak := func() {
				setTurnStamina(heavy, 30)
				config := defs.UnitConfigs[militia]
				config.Max_health = 1000
				defs.UnitConfigs[militia] = config
			}
			g := attackBackGame(t, tweak, spaceOf(0, 0, flat, placed{heavy, 0, 0, 0}, placed{militia, 1, 0, 0}, placed{heavy, 1, 1, 0}))
			foe := g.Human(1)
			attacker := unitWithID(g, foe, heavy)
			passive(g, foe, attacker.InternalID)
			setVictimBehavior(g, defs.UnitStance_STAND_GROUND, interrupt)
			g.EndTurn()
			g.Action(foe, "set-unit-behavior", gin.H{"internal_ids": []uint64{attacker.InternalID}, "stance": uint8(defs.UnitStance_DEFENSIVE), "attack_back": false})
			g.Submit(g.Human(0))
			g.Submit(foe, attackOrder(attacker, soleUnit(g, 0)))
			if attackerHurt := hurt(unitWithID(g, foe, heavy)); attackerHurt != interrupt {
				t.Fatalf("attacker hurt=%v, want %v", attackerHurt, interrupt)
			}
		})
	}
}
