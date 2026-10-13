package economy

import (
	"fmt"
	"os"
	"runtime/debug"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/config"
)

var testConfig = testconfig.Dir()

func TestMain(m *testing.M) {
	debug.SetGCPercent(800)
	if err := defs.LoadConfig(testConfig); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
