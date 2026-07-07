package reporter

type Report struct {
	SpecVersion   string           `json:"spec_version"`
	CheckedAt     string           `json:"checked_at"`
	EmulatorURL   string           `json:"emulator_url"`
	OverallStatus string           `json:"overall_status"`
	TotalChecks   int              `json:"total_checks"`
	Passed        int              `json:"passed"`
	Failed        int              `json:"failed"`
	Resources     []ResourceResult `json:"resources"`
}

type ResourceResult struct {
	Name        string        `json:"name"`
	Status      string        `json:"status"`
	TotalChecks int           `json:"total_checks"`
	Passed      int           `json:"passed"`
	Failed      int           `json:"failed"`
	Checks      []CheckDetail `json:"checks"`
}

type CheckDetail struct {
	Field     string       `json:"field"`
	Status    string       `json:"status"`
	ErrorCode string       `json:"error_code,omitempty"`
	Message   string       `json:"message"`
	Detail    *ErrorDetail `json:"detail,omitempty"`
}

type ErrorDetail struct {
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
}
