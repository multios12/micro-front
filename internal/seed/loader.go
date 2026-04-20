package seed

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func loadJSON[T any](path string) (T, error) {
	var out T
	body, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return out, fmt.Errorf("parse %s: %w", path, err)
	}
	return out, nil
}

func readSeedText(seedDir, path string) (string, error) {
	body, err := os.ReadFile(filepath.Join(seedDir, path))
	if err != nil {
		return "", err
	}
	return string(body), nil
}
