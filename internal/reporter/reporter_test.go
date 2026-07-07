package reporter_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AliseMarfina/swordfish-verifier/internal/comparator"
	"github.com/AliseMarfina/swordfish-verifier/internal/reporter"
)

func TestBuildReport_Pass(t *testing.T) {
	checks := map[string][]comparator.CheckResult{
		"Volume": {
			{
				Resource: "Volume",
				Field:    "Id",
				Status:   "PASS",
				Message:  "Field is valid",
			},
			{
				Resource: "Volume",
				Field:    "Name",
				Status:   "PASS",
				Message:  "Field is valid",
			},
		},
	}

	report := reporter.BuildReport("1.2.8", "http://localhost:5000", checks)

	if report.SpecVersion != "1.2.8" {
		t.Errorf("Expected spec version 1.2.8, got %s", report.SpecVersion)
	}

	if report.EmulatorURL != "http://localhost:5000" {
		t.Errorf("Expected emulator URL http://localhost:5000, got %s", report.EmulatorURL)
	}

	if report.OverallStatus != "PASS" {
		t.Errorf("Expected overall status PASS, got %s", report.OverallStatus)
	}

	if report.TotalChecks != 2 {
		t.Errorf("Expected 2 total checks, got %d", report.TotalChecks)
	}

	if report.Passed != 2 {
		t.Errorf("Expected 2 passed, got %d", report.Passed)
	}

	if report.Failed != 0 {
		t.Errorf("Expected 0 failed, got %d", report.Failed)
	}
}

func TestBuildReport_Fail(t *testing.T) {
	checks := map[string][]comparator.CheckResult{
		"Volume": {
			{
				Resource: "Volume",
				Field:    "Id",
				Status:   "PASS",
				Message:  "Field is valid",
			},
			{
				Resource: "Volume",
				Field:    "Name",
				Status:   "FAIL",
				Message:  "Required field is missing",
				Expected: "string",
				Actual:   "missing",
			},
		},
	}

	report := reporter.BuildReport("1.2.8", "http://localhost:5000", checks)

	if report.OverallStatus != "FAIL" {
		t.Errorf("Expected overall status FAIL, got %s", report.OverallStatus)
	}

	if report.TotalChecks != 2 {
		t.Errorf("Expected 2 total checks, got %d", report.TotalChecks)
	}

	if report.Passed != 1 {
		t.Errorf("Expected 1 passed, got %d", report.Passed)
	}

	if report.Failed != 1 {
		t.Errorf("Expected 1 failed, got %d", report.Failed)
	}

	// Check resource-level status
	if len(report.Resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(report.Resources))
	}

	if report.Resources[0].Status != "FAIL" {
		t.Errorf("Expected resource status FAIL, got %s", report.Resources[0].Status)
	}
}

func TestBuildReport_MultipleResources(t *testing.T) {
	checks := map[string][]comparator.CheckResult{
		"Volume": {
			{
				Resource: "Volume",
				Field:    "Id",
				Status:   "PASS",
				Message:  "Field is valid",
			},
		},
		"StoragePool": {
			{
				Resource: "StoragePool",
				Field:    "Name",
				Status:   "PASS",
				Message:  "Field is valid",
			},
		},
	}

	report := reporter.BuildReport("1.2.8", "http://localhost:5000", checks)

	if len(report.Resources) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(report.Resources))
	}

	if report.TotalChecks != 2 {
		t.Errorf("Expected 2 total checks, got %d", report.TotalChecks)
	}
}

func TestBuildReport_Empty(t *testing.T) {
	checks := map[string][]comparator.CheckResult{}

	report := reporter.BuildReport("1.2.8", "http://localhost:5000", checks)

	if report.OverallStatus != "FAIL" {
		t.Errorf("Expected overall status FAIL for empty report, got %s", report.OverallStatus)
	}

	if report.TotalChecks != 0 {
		t.Errorf("Expected 0 total checks, got %d", report.TotalChecks)
	}
}

func TestWriteReport(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	checks := map[string][]comparator.CheckResult{
		"Volume": {
			{
				Resource: "Volume",
				Field:    "Id",
				Status:   "PASS",
				Message:  "Field is valid",
			},
		},
	}

	report := reporter.BuildReport("1.2.8", "http://localhost:5000", checks)

	// Test writing to a file path
	filePath := filepath.Join(tmpDir, "report.json")
	if err := reporter.WriteReport(filePath, report); err != nil {
		t.Fatalf("Failed to write report: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("Report file was not created: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read report file: %v", err)
	}

	if len(content) == 0 {
		t.Errorf("Report file is empty")
	}
}

func TestWriteReport_DirectoryPath(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	checks := map[string][]comparator.CheckResult{
		"Volume": {
			{
				Resource: "Volume",
				Field:    "Id",
				Status:   "PASS",
				Message:  "Field is valid",
			},
		},
	}

	report := reporter.BuildReport("1.2.8", "http://localhost:5000", checks)

	// Test writing to a directory (should create result.json)
	if err := reporter.WriteReport(tmpDir, report); err != nil {
		t.Fatalf("Failed to write report: %v", err)
	}

	// Verify file exists
	expectedPath := filepath.Join(tmpDir, "result.json")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("Report file was not created at expected path: %v", err)
	}
}

func TestFormatErrorMessage(t *testing.T) {
	msg := reporter.FormatErrorMessage(reporter.ErrMissingRequired, "Id", "string")
	if msg == "" {
		t.Errorf("Expected error message, got empty string")
	}

	if !contains(msg, "Id") || !contains(msg, "string") {
		t.Errorf("Error message doesn't contain expected values: %s", msg)
	}
}

func TestFormatErrorMessage_UnsupportedResource(t *testing.T) {
	msg := reporter.FormatErrorMessage(reporter.ErrUnsupportedResource, "UnknownResource")
	if msg == "" {
		t.Errorf("Expected error message, got empty string")
	}

	if !contains(msg, "UnknownResource") {
		t.Errorf("Error message doesn't contain resource name: %s", msg)
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
