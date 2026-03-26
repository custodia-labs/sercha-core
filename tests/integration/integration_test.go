//go:build integration

// Package integration provides BDD integration tests using godog/Cucumber.
// These tests require Docker and are excluded from normal `go test ./...` runs.
// Run with: go test -tags=integration ./tests/integration/...
package integration

import (
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/custodia-labs/sercha-core/tests/integration/steps"
)

var opts = godog.Options{
	Output: colors.Colored(os.Stdout),
	Format: "pretty",
	Paths:  []string{"features"},
}

func init() {
	// Allow overriding format via env (e.g., GODOG_FORMAT=cucumber for JSON)
	if format := os.Getenv("GODOG_FORMAT"); format != "" {
		opts.Format = format
	}
}

func TestIntegration(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: steps.InitializeScenario,
		Options:             &opts,
	}

	if suite.Run() != 0 {
		t.Fatal("integration tests failed")
	}
}
