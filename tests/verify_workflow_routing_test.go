package tests

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

func extractCondition(t *testing.T) string {
	content, err := os.ReadFile("../.github/workflows/ci.yml")
	if err != nil {
		t.Fatalf("Failed to read ci.yml: %v", err)
	}

	re := regexp.MustCompile(`release-ready:[\s\S]*?if:\s*>-\s*([\s\S]*?)runs-on:`)
	matches := re.FindStringSubmatch(string(content))
	if len(matches) < 2 {
		t.Fatalf("Could not extract release-ready if condition")
	}
	return strings.TrimSpace(matches[1])
}

// naiveEval evaluates a subset of the GitHub Actions expression logic matching our specific if block
func naiveEval(cond string, needs map[string]string, outputs map[string]string) bool {
    // 1. replace variables
    for k, v := range needs {
        cond = strings.ReplaceAll(cond, fmt.Sprintf("needs.%s.result", k), fmt.Sprintf("'%s'", v))
    }
    for k, v := range outputs {
        cond = strings.ReplaceAll(cond, fmt.Sprintf("needs.route.outputs.%s", k), fmt.Sprintf("'%s'", v))
    }

    // Evaluate always()
    cond = strings.ReplaceAll(cond, "always()", "true")

    // The expression we have is:
    // true && ('true' == 'true' || 'false' == 'true') && ('false' == 'false' || ('success' == 'success' && 'success' == 'success')) && ('false' == 'false' || 'success' == 'success')
    // Let's implement a very basic text based evaluator

    // evaluate equality
    for {
        re := regexp.MustCompile(`'([^']+)'\s*==\s*'([^']+)'`)
        match := re.FindStringSubmatch(cond)
        if match == nil {
            break
        }
        val := "false"
        if match[1] == match[2] {
            val = "true"
        }
        cond = strings.Replace(cond, match[0], val, 1)
    }

    // evaluate parens and logical operators
    // Since we know the exact structure, we can just replace the known logical patterns

    for i := 0; i < 10; i++ { // loop to resolve inner parens
        cond = strings.ReplaceAll(cond, "true && true", "true")
        cond = strings.ReplaceAll(cond, "true && false", "false")
        cond = strings.ReplaceAll(cond, "false && true", "false")
        cond = strings.ReplaceAll(cond, "false && false", "false")

        cond = strings.ReplaceAll(cond, "true || true", "true")
        cond = strings.ReplaceAll(cond, "true || false", "true")
        cond = strings.ReplaceAll(cond, "false || true", "true")
        cond = strings.ReplaceAll(cond, "false || false", "false")

        cond = strings.ReplaceAll(cond, "(true)", "true")
        cond = strings.ReplaceAll(cond, "(false)", "false")
        cond = strings.ReplaceAll(cond, " ", "")
        cond = strings.ReplaceAll(cond, "\n", "")
        cond = strings.ReplaceAll(cond, "\r", "")

        // spaces removed...
        cond = strings.ReplaceAll(cond, "true&&true", "true")
        cond = strings.ReplaceAll(cond, "true&&false", "false")
        cond = strings.ReplaceAll(cond, "false&&true", "false")
        cond = strings.ReplaceAll(cond, "false&&false", "false")

        cond = strings.ReplaceAll(cond, "true||true", "true")
        cond = strings.ReplaceAll(cond, "true||false", "true")
        cond = strings.ReplaceAll(cond, "false||true", "true")
        cond = strings.ReplaceAll(cond, "false||false", "false")
    }

    return cond == "true"
}

func TestWorkflowRouting(t *testing.T) {
	cond := extractCondition(t)

	tests := []struct {
		name     string
		needs    map[string]string
		outputs  map[string]string
		expected bool
	}{
		{
			name:     "Ordinary pull request",
			needs:    map[string]string{"validation": "success", "lint": "success", "build": "success"},
			outputs:  map[string]string{"run_code_checks": "true", "run_build": "true", "run_release": "false", "run_publisher": "false"},
			expected: false,
		},
		{
			name:     "Normal push to main",
			needs:    map[string]string{"validation": "success", "lint": "success", "build": "success"},
			outputs:  map[string]string{"run_code_checks": "true", "run_build": "true", "run_release": "false", "run_publisher": "false"},
			expected: false,
		},
		{
			name:     "Manual build",
			needs:    map[string]string{"validation": "success", "lint": "success", "build": "success"},
			outputs:  map[string]string{"run_code_checks": "true", "run_build": "true", "run_release": "false", "run_publisher": "false"},
			expected: false,
		},
		{
			name:     "Manual release-patch or equivalent release mode",
			needs:    map[string]string{"validation": "success", "lint": "success", "build": "success"},
			outputs:  map[string]string{"run_code_checks": "true", "run_build": "true", "run_release": "true", "run_publisher": "false"},
			expected: true,
		},
		{
			name:     "A required release-critical job being skipped",
			needs:    map[string]string{"validation": "skipped", "lint": "success", "build": "success"},
			outputs:  map[string]string{"run_code_checks": "true", "run_build": "true", "run_release": "true", "run_publisher": "false"},
			expected: false,
		},
		{
			name:     "A required release-critical job failing",
			needs:    map[string]string{"validation": "failure", "lint": "success", "build": "success"},
			outputs:  map[string]string{"run_code_checks": "true", "run_build": "true", "run_release": "true", "run_publisher": "false"},
			expected: false,
		},
		{
			name:     "A required release-critical job being cancelled",
			needs:    map[string]string{"validation": "cancelled", "lint": "success", "build": "success"},
			outputs:  map[string]string{"run_code_checks": "true", "run_build": "true", "run_release": "true", "run_publisher": "false"},
			expected: false,
		},
		{
			name:     "External v* tag publication",
			needs:    map[string]string{"validation": "success", "lint": "success", "build": "success"},
			outputs:  map[string]string{"run_code_checks": "true", "run_build": "true", "run_release": "false", "run_publisher": "true"},
			expected: true,
		},
		{
			name:     "Explicit publish-tag dispatch",
			needs:    map[string]string{"validation": "skipped", "lint": "skipped", "build": "skipped"},
			outputs:  map[string]string{"run_code_checks": "false", "run_build": "false", "run_release": "false", "run_publisher": "true"},
			expected: true,
		},
		{
			name:     "A required release-critical job succeeding",
			needs:    map[string]string{"validation": "success", "lint": "success", "build": "success"},
			outputs:  map[string]string{"run_code_checks": "true", "run_build": "true", "run_release": "true", "run_publisher": "false"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := naiveEval(cond, tt.needs, tt.outputs)
			if actual != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, actual)
			}
		})
	}
}
