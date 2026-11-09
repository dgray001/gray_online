package experiment

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func sourceHash() (string, error) {
	hash := sha256.New()
	for _, root := range []string{"neural", "../backend/game/games/fiddlesticks", "../backend/game/game_utils", "../backend/util"} {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			fmt.Fprintln(hash, path)
			hash.Write(data)
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
