package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/AliseMarfina/swordfish-verifier/internal/comparator"
	"github.com/AliseMarfina/swordfish-verifier/internal/config"
	"github.com/AliseMarfina/swordfish-verifier/internal/reporter"
	"github.com/AliseMarfina/swordfish-verifier/parser/model"
)

func main() {
	configPath := flag.String("config", "config.yaml", "путь к файлу конфигурации")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}
	log.Printf("Конфигурация загружена: эмулятор=%s, таймаут=%d", cfg.EmulatorURL, cfg.Timeout)

	data, err := os.ReadFile("parsed_spec.json")
	if err != nil {
		log.Fatalf("Не удалось прочитать parsed_spec.json: %v", err)
	}
	var spec model.Spec
	if err := json.Unmarshal(data, &spec); err != nil {
		log.Fatalf("Ошибка разбора parsed_spec.json: %v", err)
	}

	checksPerResource := make(map[string][]comparator.CheckResult)

	validJSON := []byte(`{
		"Id": "vol-1",
		"Name": "MyVolume",
		"RAIDType": "RAID10",
		"CapacityBytes": 1073741824,
		"VolumeType": "StripedWithParity",
		"Encrypted": false,
		"Links": {
			"Drives": [ {"@odata.id": "/redfish/v1/Drives/1"} ]
		}
	}`)
	validChecks, err := comparator.Compare("Volume", &spec, validJSON)
	if err != nil {
		log.Printf("Ошибка при проверке валидного ответа: %v", err)
	} else {
		checksPerResource["Volume (валидный)"] = validChecks
	}

	invalidJSON := []byte(`{
		"Id": 123,
		"Name": "BadVolume",
		"RAIDType": "RAID7",
		"CapacityBytes": "not a number",
		"VolumeType": "RawDevice",
		"Encrypted": false,
		"UnknownField": "лишнее поле"
	}`)
	invalidChecks, err := comparator.Compare("Volume", &spec, invalidJSON)
	if err != nil {
		log.Printf("Ошибка при проверке ответа с ошибками: %v", err)
	} else {
		checksPerResource["Volume (с ошибками)"] = invalidChecks
	}

	report := reporter.BuildReport(spec.SourceVersion, cfg.EmulatorURL, checksPerResource)

	outputPath := "result.json"
	if cfg.OutputPath != "" {
		outputPath = cfg.OutputPath
	}
	if err := reporter.WriteReport(outputPath, report); err != nil {
		log.Fatalf("Ошибка записи отчёта: %v", err)
	}
	log.Printf("Отчёт сохранён в %s", outputPath)
}
