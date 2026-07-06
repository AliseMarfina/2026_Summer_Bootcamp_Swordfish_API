package formats

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/AliseMarfina/swordfish-verifier/parser/model"
	"github.com/AliseMarfina/swordfish-verifier/parser/v2universal"
)

type PDFFormat struct{}

func (PDFFormat) Name() string { return "pdf" }

func (PDFFormat) Supports(path string) bool {
	ok, _ := dirOrFileHasSuffix(path, ".pdf")
	return ok
}

var (
	headingRe  = regexp.MustCompile(`(?m)^\s*9\.5\.\d+\s+([A-Za-z][A-Za-z0-9]*)(?:\s+([0-9]+(?:\.[0-9]+)*))?\s*$`)
	subHeadRe  = regexp.MustCompile(`(?m)^\s*9\.5\.\d+\.\d+(?:\.\d+)?\s+\S`)
	urisHeadRe = regexp.MustCompile(`(?m)^\s*9\.5\.\d+\.\d+\s+URIs\s*$`)
)

func (PDFFormat) Parse(path string, resourceFilter []string) (*model.Spec, error) {
	files, err := listFilesLocal(path, ".pdf")
	if err != nil {
		return nil, err
	}
	filter := make(map[string]bool, len(resourceFilter))
	for _, r := range resourceFilter {
		filter[r] = true
	}

	spec := model.NewSpec()
	for _, file := range files {
		text, err := extractLayoutText(file)
		if err != nil {
			return nil, fmt.Errorf("pdf: %w", err)
		}
		spec.Sources = append(spec.Sources, file)

		headings := headingRe.FindAllStringSubmatchIndex(text, -1)
		for i, h := range headings {
			name := text[h[2]:h[3]]
			version := ""
			if h[4] >= 0 {
				version = text[h[4]:h[5]]
			}
			if len(filter) > 0 && !filter[name] {
				continue
			}
			sectionStart := h[1]
			sectionEnd := len(text)
			if i+1 < len(headings) {
				sectionEnd = headings[i+1][0]
			}
			chunk := text[sectionStart:sectionEnd]

			resource := &model.Resource{
				Name:       name,
				Version:    version,
				Properties: map[string]*model.Property{},
				Endpoints:  extractURIs(chunk),
				SpecRef:    fmt.Sprintf("%s#%s", file, strings.TrimSpace(headingLine(text, h[0]))),
			}
			spec.Resources[name] = resource
		}
	}
	return spec, nil
}

func extractURIs(chunk string) []string {
	loc := urisHeadRe.FindStringIndex(chunk)
	if loc == nil {
		return nil
	}
	rest := chunk[loc[1]:]
	if next := subHeadRe.FindStringIndex(rest); next != nil {
		rest = rest[:next[0]]
	}

	var uris []string
	for _, line := range strings.Split(rest, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "/redfish/v1/") {
			uris = append(uris, trimmed)
			continue
		}
		if len(uris) > 0 {
			uris[len(uris)-1] += trimmed
		}
	}
	return uris
}

func headingLine(text string, offset int) string {
	end := strings.IndexByte(text[offset:], '\n')
	if end < 0 {
		return text[offset:]
	}
	return text[offset : offset+end]
}

func extractLayoutText(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	cmd := exec.Command("pdftotext", "-layout", path, "-")
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("running pdftotext (poppler-utils must be installed): %s: %w", stderr.String(), err)
	}
	return out.String(), nil
}

func init() {
	v2universal.Register(PDFFormat{})
}
