package reporter

import (
	"strings"
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

	if report.TotalChecks == 0 {
		report.OverallStatus = "FAIL"
	} else if report.Failed == 0 {
		report.OverallStatus = "PASS"
	} else {
		report.OverallStatus = "FAIL"
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
		if d.Status == "FAIL" {
			res.Failed++
			res.Status = "FAIL"
		} else {
			res.Passed++
		}
	}
	return res
}

func convertCheck(c comparator.CheckResult) CheckDetail {
	d := CheckDetail{
		Field:  c.Field,
		Status: c.Status,
	}

	switch c.Status {
	case "FAIL":
		msg := c.Message
		switch {
		case strings.Contains(msg, "не найден в спецификации"):
			d.ErrorCode = string(ErrUnsupportedResource)
			d.Message = FormatErrorMessage(ErrUnsupportedResource, c.Resource)
		case strings.Contains(msg, "Обязательное поле"):
			d.ErrorCode = string(ErrMissingRequired)
			d.Message = c.Message
		case strings.Contains(msg, "Несовпадение типа"):
			d.ErrorCode = string(ErrTypeMismatch)
			d.Message = c.Message
		case strings.Contains(msg, "Недопустимое значение"):
			d.ErrorCode = string(ErrInvalidEnumValue)
			d.Message = c.Message
		case strings.Contains(msg, "не может быть null"):
			d.ErrorCode = string(ErrNullableViolation)
			d.Message = c.Message
		case strings.Contains(msg, "не предусмотрено спецификацией") || strings.Contains(msg, "не предусмотрено во вложенном"):
			d.ErrorCode = string(ErrUnexpectedField)
			d.Message = c.Message
		default:
			d.ErrorCode = string(ErrUnknown)
			d.Message = c.Message
		}
		d.Detail = &ErrorDetail{
			Expected: c.Expected,
			Actual:   c.Actual,
		}

	case "PASS":
		d.Message = c.Message
		if c.Expected != "" || c.Actual != "" {
			d.Detail = &ErrorDetail{
				Expected: c.Expected,
				Actual:   c.Actual,
			}
		}
	}

	return d
}
