package behavior

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func matrix(output string, summary Summary) error {
	var text strings.Builder
	fmt.Fprint(&text, `<!doctype html><html lang="en"><meta charset="utf-8"><title>Checkpoint behavior</title><style>body{font:14px monospace;margin:24px}table{border-collapse:collapse;margin-bottom:40px}th,td{border:1px solid #ccc;padding:5px;text-align:right}th{white-space:nowrap}.column{writing-mode:vertical-rl;transform:rotate(180deg);height:205px;text-align:left}td{min-width:45px}h1,h2,p{font-family:system-ui}</style><h1>Checkpoint behavior on identical situations</h1><p>Lower values mean more similar behavior. Hover a cell for exact values. Each table uses its own color scale.</p>`)
	matrixTable(&text, summary, false)
	matrixTable(&text, summary, true)
	fmt.Fprint(&text, "</html>")
	return os.WriteFile(filepath.Join(output, "matrix.html"), []byte(text.String()), 0600)
}
