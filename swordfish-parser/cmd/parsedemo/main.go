package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/AliseMarfina/swordfish-verifier/parser/v1json"
	"github.com/AliseMarfina/swordfish-verifier/parser/v2universal"
	_ "github.com/AliseMarfina/swordfish-verifier/parser/v2universal/formats"
)

func main() {
	mode := flag.String("mode", "v2", "which parser version to run: v1 (JSON only) or v2 (universal)")
	schemaDir := flag.String("schema", "", "directory of Redfish/Swordfish JSON Schema files")
	xmlPath := flag.String("xml", "", "CSDL metadata XML file or directory (v2 only)")
	pdfPath := flag.String("pdf", "", "Swordfish specification PDF file (v2 only)")
	overridePath := flag.String("override", "", "YAML override/augmentation file (v2 only)")
	resources := flag.String("resources", "", "comma-separated resource filter, e.g. Volume,StoragePool")
	flag.Parse()

	var filter []string
	if *resources != "" {
		filter = splitCSV(*resources)
	}

	switch *mode {
	case "v1":
		if *schemaDir == "" {
			log.Fatal("-schema is required in v1 mode")
		}
		spec, err := v1json.Parse(v1json.Config{SchemaDir: *schemaDir, ResourceFilter: filter})
		if err != nil {
			log.Fatalf("parse failed: %v", err)
		}
		printSpec(spec)

	case "v2":
		var sources []v2universal.Source
		if *schemaDir != "" {
			sources = append(sources, v2universal.Source{Path: *schemaDir, Format: "json"})
		}
		if *xmlPath != "" {
			sources = append(sources, v2universal.Source{Path: *xmlPath, Format: "xml"})
		}
		if *pdfPath != "" {
			sources = append(sources, v2universal.Source{Path: *pdfPath, Format: "pdf"})
		}
		if *overridePath != "" {
			sources = append(sources, v2universal.Source{Path: *overridePath, Format: "yaml"})
		}
		if len(sources) == 0 {
			log.Fatal("at least one of -schema, -xml, -pdf, -override is required in v2 mode")
		}
		spec, err := v2universal.Parse(v2universal.Config{Sources: sources, ResourceFilter: filter})
		if err != nil {
			log.Fatalf("parse failed: %v", err)
		}
		printSpec(spec)

	default:
		log.Fatalf("unknown -mode %q (want v1 or v2)", *mode)
	}
}

func printSpec(spec interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(spec); err != nil {
		fmt.Fprintln(os.Stderr, "encode error:", err)
		os.Exit(1)
	}
}

func splitCSV(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			if i > start {
				out = append(out, s[start:i])
			}
			start = i + 1
		}
	}
	return out
}
