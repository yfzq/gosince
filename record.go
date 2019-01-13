package gosince

import "time"

// APIRecord represents record for an golang API
type APIRecord struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Version     string `json:"version"`
	PackageName string `json:"package_name"`
	Description string `json:"description"`
	GolangURL   string `json:"golang_url"`
}

// DBTimeout is the timeout for any sqlite operations.
const DBTimeout = 5 * time.Second
