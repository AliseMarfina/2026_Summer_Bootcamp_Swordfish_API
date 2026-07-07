package main

import (
	"encoding/json"
	"flag"
	"log"
	"strings"

	"github.com/AliseMarfina/swordfish-verifier/internal/client"
	"github.com/AliseMarfina/swordfish-verifier/internal/comparator"
	"github.com/AliseMarfina/swordfish-verifier/internal/config"
	"github.com/AliseMarfina/swordfish-verifier/internal/parser/v2universal"
	_ "github.com/AliseMarfina/swordfish-verifier/internal/parser/v2universal/formats"
	"github.com/AliseMarfina/swordfish-verifier/internal/reporter"
)

func main() {
	// 1. Загружаем конфиг
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Не удалось загрузить конфиг: %v", err)
	}

	log.Printf("Загружен конфиг: эмулятор=%s, таймаут=%d", cfg.EmulatorURL, cfg.Timeout)

	// 2. Загружаем и парсим спецификацию (схемы)
	log.Println("Загрузка спецификации Swordfish...")
	var sources []v2universal.Source
	if cfg.SpecPath != "" {
		// Парсер сам определит формат по расширению (json, xml, yaml и т.д.)
		sources = append(sources, v2universal.Source{Path: cfg.SpecPath})
	}
	// Если в конфиге есть путь к файлу оверрайдов, добавляем его
	// (здесь предполагается, что у вас есть поле OverridePath в структуре Config)
	// if cfg.OverridePath != "" {
	// 	sources = append(sources, v2universal.Source{Path: cfg.OverridePath, Format: "yaml"})
	// }

	if len(sources) == 0 {
		log.Fatal("В конфиге не указан путь к спецификации (specification_path)")
	}

	spec, err := v2universal.Parse(v2universal.Config{
		Sources:        sources,
		ResourceFilter: cfg.ResourcesFilter,
	})
	if err != nil {
		log.Fatalf("Ошибка парсинга спецификации: %v", err)
	}
	log.Printf("Спецификация загружена. Всего ресурсов: %d", len(spec.Resources))

	// 3. Создаём HTTP-клиент для связи с эмулятором
	log.Println("Инициализация клиента...")
	c, err := client.NewClient(cfg)
	if err != nil {
		log.Fatalf("Ошибка создания клиента: %v", err)
	}

	// 4. Получаем список эндпоинтов для проверки (автообход или ручной список)
	log.Println("Сбор эндпоинтов эмулятора...")
	endpoints, err := c.GetEndpoints(cfg.ResourcesFilter)
	if err != nil {
		log.Fatalf("Ошибка при обходе эндпоинтов: %v", err)
	}
	log.Printf("Найдено %d эндпоинтов для проверки", len(endpoints))

	// 5. Валидируем каждый эндпоинт
	checksPerResource := make(map[string][]comparator.CheckResult)

	for _, ep := range endpoints {
		log.Printf("Проверка эндпоинта: %s", ep)

		responseBytes, status, err := c.Get(ep)
		if err != nil {
			log.Printf("Ошибка запроса к %s: %v", ep, err)
			continue
		}
		if status != 200 {
			log.Printf("Пропуск %s: статус %d", ep, status)
			continue
		}

		// Определяем имя ресурса по полю @odata.type из ответа
		var respData map[string]interface{}
		if err := json.Unmarshal(responseBytes, &respData); err != nil {
			log.Printf("Ошибка парсинга JSON из %s: %v", ep, err)
			continue
		}
		odataType, ok := respData["@odata.type"].(string)
		if !ok {
			log.Printf("Поле @odata.type не найдено в ответе %s", ep)
			continue
		}
		resourceName := extractResourceName(odataType)
		if resourceName == "" {
			log.Printf("Не удалось извлечь имя ресурса из %s", odataType)
			continue
		}

		// Запускаем сравнение
		results, err := comparator.Compare(resourceName, spec, responseBytes)
		if err != nil {
			log.Printf("Ошибка сравнения для %s: %v", ep, err)
			continue
		}

		// Сохраняем результаты, сгруппированные по ресурсу
		checksPerResource[resourceName] = append(checksPerResource[resourceName], results...)
	}

	// 6. Генерируем итоговый отчёт
	log.Println("Генерация отчёта...")
	report := reporter.BuildReport(spec.SourceVersion, cfg.EmulatorURL, checksPerResource)

	if err := reporter.WriteReport(cfg.OutputPath, report); err != nil {
		log.Fatalf("Ошибка записи отчёта: %v", err)
	}

	log.Printf("✅ Готово! Отчёт сохранён в: %s", cfg.OutputPath)
}

// extractResourceName извлекает имя ресурса из строки @odata.type.
// Пример: "#Volume.v1_6_0.Volume" -> "Volume"
func extractResourceName(odataType string) string {
	parts := strings.Split(odataType, ".")
	if len(parts) == 0 {
		return ""
	}
	base := strings.TrimPrefix(parts[0], "#")
	return base
}
