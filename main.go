package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/AliseMarfina/swordfish-verifier/internal/client"
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
<<<<<<< Updated upstream
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}
	log.Printf("Конфигурация загружена: эмулятор=%s, таймаут=%d", cfg.EmulatorURL, cfg.Timeout)
=======
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Config loaded: emulator=%s, timeout=%d", cfg.EmulatorURL, cfg.Timeout)
>>>>>>> Stashed changes

	// Load specification
	specData, err := os.ReadFile("parsed_spec.json")
	if err != nil {
		log.Fatalf("Failed to read parsed_spec.json: %v", err)
	}
	var spec model.Spec
<<<<<<< Updated upstream
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
=======
	if err := json.Unmarshal(specData, &spec); err != nil {
		log.Fatalf("Failed to parse parsed_spec.json: %v", err)
	}

	// Create HTTP client
	httpClient := client.NewClient(cfg)

	// Ping server to check connectivity
	if err := httpClient.Ping(); err != nil {
		log.Fatalf("Failed to connect to server at %s: %v", cfg.EmulatorURL, err)
	}
	log.Printf("Successfully connected to server at %s", cfg.EmulatorURL)

	// Test all resources from the spec
	checksPerResource := make(map[string][]comparator.CheckResult)
	resources := getResourcesToTest(cfg, spec)

	for _, resourceName := range resources {
		resource, ok := spec.Resources[resourceName]
		if !ok {
			log.Printf("Resource %s not found in specification", resourceName)
			continue
		}

		results := testResourceEndpoints(httpClient, resourceName, resource, &spec)
		checksPerResource[resourceName] = results
	}

	// Build report
	report := reporter.BuildReport(spec.SourceVersion, cfg.EmulatorURL, checksPerResource)

	// Write report to file
	if err := reporter.WriteReport(cfg.OutputPath, report); err != nil {
		log.Fatalf("Failed to write report: %v", err)
	}

	log.Printf("Report saved to %s", cfg.OutputPath)
	log.Printf("Overall status: %s (Passed: %d, Failed: %d)", report.OverallStatus, report.Passed, report.Failed)
}

// getResourcesToTest returns the list of resources to test based on config
func getResourcesToTest(cfg *config.Config, spec model.Spec) []string {
	if len(cfg.ResourcesFilter) > 0 {
		return cfg.ResourcesFilter
	}

	// If no filter specified, test all resources
	var resources []string
	for name := range spec.Resources {
		resources = append(resources, name)
	}
	return resources
}

// testResourceEndpoints tests all endpoints for a given resource
func testResourceEndpoints(httpClient *client.Client, resourceName string, resource *model.Resource, spec *model.Spec) []comparator.CheckResult {
	var allResults []comparator.CheckResult

	if len(resource.Endpoints) == 0 {
		log.Printf("No endpoints defined for resource %s in specification", resourceName)
		return allResults
	}

	// Test the first endpoint for initial validation
	endpoint := resource.Endpoints[0]
	log.Printf("Testing resource %s on endpoint %s", resourceName, endpoint)

	body, err := httpClient.GetResource(endpoint)
	if err != nil {
		log.Printf("Error fetching resource %s: %v", resourceName, err)
		allResults = append(allResults, comparator.CheckResult{
			Resource: resourceName,
			Status:   "FAIL",
			Message:  "Failed to fetch resource: " + err.Error(),
		})
		return allResults
	}

	// Compare response with specification
	results, err := comparator.Compare(resourceName, spec, body)
	if err != nil {
		log.Printf("Error comparing resource %s: %v", resourceName, err)
		allResults = append(allResults, comparator.CheckResult{
			Resource: resourceName,
			Status:   "FAIL",
			Message:  "Failed to compare with specification: " + err.Error(),
		})
		return allResults
	}

	allResults = append(allResults, results...)
	return allResults
>>>>>>> Stashed changes
}
