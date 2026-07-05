package main

import (
	"flag"
	"log"
	"swordfish-verifier/internal/config"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Loaded config: emulator=%s, timeout=%d", cfg.EmulatorURL, cfg.Timeout)
	// Здесь дальше будем вызывать другие модули
}