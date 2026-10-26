package ai

import (
	"fmt"
	"strings"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func (l *varNameLog) noteConfigRead(ref string) {
	table, field, _ := strings.Cut(ref, ".")
	if !defs.HasConfigField(table, field) {
		l.errors = append(l.errors, fmt.Sprintf("config(%s, %s, ...): unknown table or field", table, field))
	}
}
