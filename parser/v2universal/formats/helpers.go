package formats

import (
	"os"
	"path/filepath"
	"strings"
)

func dirOrFileHasSuffix(path string, suffix string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return strings.HasSuffix(strings.ToLower(path), suffix), nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), suffix) {
			return true, nil
		}
	}
	return false, nil
}

func schemaDirFor(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return path, nil
	}
	return filepath.Dir(path), nil
}
