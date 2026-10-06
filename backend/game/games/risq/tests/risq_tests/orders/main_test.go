package orders

import (
	"fmt"
	"os"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

const shippedConfig = "../../../config"

func TestMain(m *testing.M) {
	if err := defs.LoadConfig(shippedConfig); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
