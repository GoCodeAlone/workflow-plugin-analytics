package internal

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIInjectUsesTagIDEnv(t *testing.T) {
	dir := t.TempDir()
	htmlPath := filepath.Join(dir, "index.html")
	if err := os.WriteFile(htmlPath, []byte(sampleHTML), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOOGLE_TAG_ID", "G-ENV123")

	var stdout, stderr bytes.Buffer
	code := newCLIProvider(&stdout, &stderr).RunCLI([]string{
		"analytics", "inject",
		"--provider", "google-analytics",
		"--tag-id-env", "GOOGLE_TAG_ID",
		"--html", htmlPath,
	})
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "G-ENV123") {
		t.Fatalf("tag id was not injected:\n%s", string(data))
	}
	if !strings.Contains(stdout.String(), "changed 1") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestCLIInjectEmptyEnvNoop(t *testing.T) {
	dir := t.TempDir()
	htmlPath := filepath.Join(dir, "index.html")
	if err := os.WriteFile(htmlPath, []byte(sampleHTML), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOOGLE_TAG_ID", "")

	var stdout, stderr bytes.Buffer
	code := newCLIProvider(&stdout, &stderr).RunCLI([]string{
		"analytics", "inject",
		"--provider", "google-analytics",
		"--tag-id-env", "GOOGLE_TAG_ID",
		"--html", htmlPath,
	})
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "skipped (empty tag id)") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestCLIAnalyticsGoogleGA4EnsureDryRun(t *testing.T) {
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)
	var stdout, stderr bytes.Buffer
	code := newCLIProvider(&stdout, &stderr).RunCLI([]string{
		"analytics", "google", "ga4", "ensure",
		"--account", "accounts/123",
		"--property-name", "example.com",
		"--stream-name", "example.com",
		"--default-uri", "https://example.com",
		"--dry-run",
	})
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	var result GA4EnsureResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("json output: %v\n%s", err, stdout.String())
	}
	if !result.DryRun {
		t.Fatalf("dry_run=false: %#v", result)
	}
	if got := operationNames(result.Operations); !sameStrings(got, []string{"create_property", "create_web_data_stream"}) {
		t.Fatalf("operations = %v", got)
	}
	auditPath := filepath.Join(stateHome, "wfctl", "plugins", "analytics", "google-audit.jsonl")
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("audit file: %v", err)
	}
	if !strings.Contains(string(data), `"action":"ga4.ensure"`) {
		t.Fatalf("audit missing ga4 action: %s", string(data))
	}
}

func TestCLIAnalyticsGoogleGTMEnsureDryRun(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	var stdout, stderr bytes.Buffer
	code := newCLIProvider(&stdout, &stderr).RunCLI([]string{
		"analytics", "google", "gtm", "ensure",
		"--account", "accounts/456",
		"--container-name", "example.com",
		"--domain", "example.com",
		"--workspace-name", "workflow",
		"--measurement-id", "G-ABC123",
		"--dry-run",
	})
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	var result GTMEnsureResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("json output: %v\n%s", err, stdout.String())
	}
	if !result.DryRun {
		t.Fatalf("dry_run=false: %#v", result)
	}
	if got := operationNames(result.Operations); !sameStrings(got, []string{"create_container", "create_workspace", "create_gtag_config"}) {
		t.Fatalf("operations = %v", got)
	}
}
