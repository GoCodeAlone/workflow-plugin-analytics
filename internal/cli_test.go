package internal

import (
	"bytes"
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
