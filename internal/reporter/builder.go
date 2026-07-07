package reporter

import (
	"time"

	"github.com/AliseMarfina/swordfish-verifier/internal/comparator"
)

func BuildReport(specVersion, emulatorURL string, checksPerResource map[string][]comparator.CheckResult) *Report {
	report := &Report{
		SpecVersion: specVersion,
		CheckedAt:   time.Now().UTC().Format(time.RFC3339),
		EmulatorURL: emulatorURL,
	}

	for resourceKey, checks := range checksPerResource {
		res := convertResource(resourceKey, checks)
		report.Resources = append(report.Resources, res)

		report.TotalChecks += res.TotalChecks
		report.Passed += res.Passed
		report.Failed += res.Failed
	}

	// Итоговый статус
	if report.TotalChecks == 0 || report.Failed > 0 {
		report.OverallStatus = "FAIL"
	} else {
		report.OverallStatus = "PASS"
	}

	return report

}

func convertResource(resourceKey string, checks []comparator.CheckResult) ResourceResult {
	res := ResourceResult{
		Name:   resourceKey,
		Status: "PASS",
	}

	for _, c := range checks {
		d := convertCheck(c)
		res.Checks = append(res.Checks, d)

		res.TotalChecks++

		switch c.Status {
		case "FAIL":
			res.Failed++
			res.Status = "FAIL"
		case "PASS":
			res.Passed++
		case "NOT_SUPPORTED":
			res.Failed++
			res.Status = "FAIL"
		}
	}

	return res

}

func convertCheck(c comparator.CheckResult) CheckDetail {
	d := CheckDetail{
		Field:     c.Field,
		Status:    c.Status,
		Message:   c.Message,
		ErrorCode: c.ErrorCode,
	}

	// Детали (если есть)
	if c.Expected != "" || c.Actual != "" {
		d.Detail = &ErrorDetail{
			Expected: c.Expected,
			Actual:   c.Actual,
		}
	}

	// Обработка NOT_SUPPORTED (если вдруг без ErrorCode)
	if c.Status == "NOT_SUPPORTED" && d.ErrorCode == "" {
		d.ErrorCode = string(ErrUnsupportedResource)
	}

	return d

}
