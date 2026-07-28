// Package eval provides the AI chatbot evaluation harness.
//
// It runs a corpus of routing cases against the supervisor and scores whether
// the supervisor routes to the expected specialist(s). This is the measurement
// layer that turns "the AI doesn't do what I want" from a guessing game into a
// regression-tested decision: every change to the supervisor prompt or routing
// config runs against this corpus.
//
// The pure corpus/CheckRoute types live here (reusable, no ai-package coupling).
// The actual supervisor execution lives in eval_test.go (package ai) so it can
// use the unexported test helpers (newFakeProvider etc.) without bloating the
// ai package's exported API.
package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Case is one routing eval case.
type Case struct {
	ID                    string   `json:"id"`
	Message               string   `json:"message"`
	PageContext           string   `json:"page_context"`
	ExpectedRouteIncludes []string `json:"expected_route_includes"`
	ExpectedRouteExcludes []string `json:"expected_route_excludes"`
	Category              string   `json:"category"`
	Notes                 string   `json:"notes"`
	// DeterministicPlan is the supervisor-plan JSON the fake provider returns
	// in deterministic mode. Required for the deterministic test; absent in
	// live mode (the real LLM produces the plan).
	DeterministicPlan string `json:"deterministic_plan,omitempty"`
}

// Corpus is the loaded eval file.
type Corpus struct {
	Comment string `json:"_comment"`
	Cases   []Case `json:"cases"`
}

// LoadCorpus reads a JSON eval corpus from disk.
func LoadCorpus(path string) (*Corpus, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read corpus: %w", err)
	}
	var c Corpus
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse corpus: %w", err)
	}
	return &c, nil
}

// Result is the outcome of one case.
type Result struct {
	Case   Case
	Passed bool
	Reason string
	Route  []string
}

// CheckRoute scores a produced route against the case's expectations.
// Returns whether it passed and a human-readable reason.
func CheckRoute(c Case, route []string) (bool, string) {
	routeSet := make(map[string]bool, len(route))
	for _, r := range route {
		routeSet[r] = true
	}
	for _, want := range c.ExpectedRouteIncludes {
		if !routeSet[want] {
			return false, fmt.Sprintf("expected route to include %q, got %v", want, route)
		}
	}
	for _, dont := range c.ExpectedRouteExcludes {
		if routeSet[dont] {
			return false, fmt.Sprintf("expected route to exclude %q, got %v", dont, route)
		}
	}
	return true, "ok"
}

// SummaryReport renders a compact pass/fail summary string.
func SummaryReport(results []Result) string {
	var sb strings.Builder
	passed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		}
	}
	fmt.Fprintf(&sb, "Eval: %d/%d passed\n", passed, len(results))
	for _, r := range results {
		mark := "✓"
		if !r.Passed {
			mark = "✗"
		}
		fmt.Fprintf(&sb, "  %s %s [%s] → route=%v\n", mark, r.Case.ID, r.Case.Category, r.Route)
		if !r.Passed {
			fmt.Fprintf(&sb, "      %s\n", r.Reason)
		}
	}
	return sb.String()
}

// CountPassed returns the number of passing results.
func CountPassed(results []Result) int {
	n := 0
	for _, r := range results {
		if r.Passed {
			n++
		}
	}
	return n
}
