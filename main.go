package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/AliseMarfina/swordfish-verifier/internal/comparator"
	"github.com/AliseMarfina/swordfish-verifier/internal/config"
	"github.com/AliseMarfina/swordfish-verifier/parser/model"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Не удалось загрузить конфиг: %v", err)
	}

	log.Printf("Загружен конфиг: emulator=%s, timeout=%d", cfg.EmulatorURL, cfg.Timeout)

	// Загружаем заранее подготовленный spec из файла parsed_spec.json
	data, err := os.ReadFile("parsed_spec.json")
	if err != nil {
		log.Fatalf("Не удалось прочитать parsed_spec.json: %v", err)
	}
	var spec model.Spec
	if err := json.Unmarshal(data, &spec); err != nil {
		log.Fatalf("Не удалось распознать parsed_spec.json: %v", err)
	}
	jsonResponse := []byte(`{"Id": "some-volume-id", "Name": "test-volume"}`)
	results, err := comparator.Compare("Volume", &spec, jsonResponse)
	if err != nil {
		log.Fatalf("Comparison error: %v", err)
	}
	log.Printf("Results: %+v", results)
}
