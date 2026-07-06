package v2universal

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AliseMarfina/swordfish-verifier/parser/model"
)

type Source struct {
	Path string

	Format string
}

type Config struct {
	Sources []Source

	ResourceFilter []string

	SpecVersion string
}

type FormatParser interface {
	Name() string

	Supports(path string) bool

	Parse(path string, resourceFilter []string) (*model.Spec, error)
}

var registry []FormatParser

func Register(p FormatParser) {
	registry = append([]FormatParser{p}, registry...)
}

func Parse(cfg Config) (*model.Spec, error) {
	if len(cfg.Sources) == 0 {
		return nil, fmt.Errorf("v2universal: at least one Source is required")
	}

	final := model.NewSpec()
	final.SourceVersion = cfg.SpecVersion

	for _, src := range cfg.Sources {
		fp, err := resolveFormatParser(src)
		if err != nil {
			return nil, err
		}
		fragment, err := fp.Parse(src.Path, cfg.ResourceFilter)
		if err != nil {
			return nil, fmt.Errorf("v2universal: %s parser failed on %s: %w", fp.Name(), src.Path, err)
		}
		final.Merge(fragment)
	}

	if len(cfg.ResourceFilter) > 0 {
		applyFilter(final, cfg.ResourceFilter)
	}

	return final, nil
}

func resolveFormatParser(src Source) (FormatParser, error) {
	if src.Format != "" {
		for _, fp := range registry {
			if fp.Name() == src.Format {
				return fp, nil
			}
		}
		return nil, fmt.Errorf("v2universal: no format parser registered for forced format %q", src.Format)
	}
	for _, fp := range registry {
		if fp.Supports(src.Path) {
			return fp, nil
		}
	}
	return nil, fmt.Errorf("v2universal: no format parser recognizes %q (tried: %s)", src.Path, formatNames())
}

func formatNames() string {
	names := make([]string, 0, len(registry))
	for _, fp := range registry {
		names = append(names, fp.Name())
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func applyFilter(spec *model.Spec, filter []string) {
	keep := make(map[string]bool, len(filter))
	for _, name := range filter {
		keep[name] = true
	}
	for name := range spec.Resources {
		if !keep[name] {
			delete(spec.Resources, name)
		}
	}
}

func listFiles(dir string, exts ...string) ([]string, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if hasAnyExt(dir, exts) {
			return []string{dir}, nil
		}
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		full := filepath.Join(dir, e.Name())
		if hasAnyExt(full, exts) {
			files = append(files, full)
		}
	}
	return files, nil
}

func hasAnyExt(path string, exts []string) bool {
	lower := strings.ToLower(path)
	for _, ext := range exts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}
