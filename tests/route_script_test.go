package tests

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func extractRouteScript(t *testing.T) string {
	content, err := os.ReadFile("../.github/workflows/ci.yml")
	if err != nil {
		t.Fatalf("Failed to read ci.yml: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	var script []string
	inScript := false

	for _, line := range lines {
		if strings.Contains(line, "id: route") {
			continue
		}
		if !inScript && strings.Contains(line, "run: |") {
			inScript = true
			continue
		}

		if inScript {
		    trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(line, "          ") { // Script indentation is 10 spaces
				script = append(script, line)
			} else {
				break
			}
		}
	}

	return strings.Join(script, "\n")
}

func testRouteExecution(t *testing.T, env map[string]string) map[string]string {
	script := extractRouteScript(t)

	// Write to temp file
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "route.sh")
	outPath := filepath.Join(tmpDir, "github_output")

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}
	if err := os.WriteFile(outPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write output file: %v", err)
	}

	cmd := exec.Command("bash", scriptPath)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("GITHUB_OUTPUT=%s", outPath))
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	out, err := cmd.CombinedOutput()

	// Read outputs
	outputs := make(map[string]string)
	if err != nil {
	    outputs["error"] = fmt.Sprintf("Command failed: %v. Output: %s", err, string(out))
	} else {
	outFile, err := os.Open(outPath)
	if err == nil {
		scanner := bufio.NewScanner(outFile)
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				outputs[parts[0]] = parts[1]
			}
		}
		outFile.Close()
	}
	}

	return outputs
}

func TestRoutePublishTagSuccess(t *testing.T) {
	env := map[string]string{
		"EVENT_NAME": "workflow_dispatch",
		"INPUT_MODE": "publish-tag",
		"GITHUB_REF": "refs/tags/v1.0.0",
		"REF_TYPE":   "tag",
	}

	outputs := testRouteExecution(t, env)

	if outputs["error"] != "" {
	    t.Fatalf("Script failed: %s", outputs["error"])
	}

	if outputs["run_code_checks"] != "true" {
		t.Errorf("Expected run_code_checks=true, got %s", outputs["run_code_checks"])
	}
	if outputs["run_build"] != "true" {
		t.Errorf("Expected run_build=true, got %s", outputs["run_build"])
	}
	if outputs["run_publisher"] != "true" {
		t.Errorf("Expected run_publisher=true, got %s", outputs["run_publisher"])
	}
}

func TestRoutePublishTagFailure(t *testing.T) {
	env := map[string]string{
		"EVENT_NAME": "workflow_dispatch",
		"INPUT_MODE": "publish-tag",
		"GITHUB_REF": "refs/heads/main",
		"REF_TYPE":   "branch",
	}

	outputs := testRouteExecution(t, env)

	if outputs["error"] == "" {
	    t.Fatalf("Expected script to fail on invalid ref, but it succeeded")
	}
}
