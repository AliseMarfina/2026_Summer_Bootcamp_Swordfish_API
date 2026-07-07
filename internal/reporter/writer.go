package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func WriteReport(outputPath string, report *Report) error {
	info, err := os.Stat(outputPath)
	if err == nil && info.IsDir() {
		outputPath = filepath.Join(outputPath, "result.json")
	}

	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("не удалось создать директорию для отчёта: %w", err)
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации отчёта: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("ошибка записи файла отчёта: %w", err)
	}

	return nil
}
