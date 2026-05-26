package internal

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func TestPluginExposesGoogleProviderModule(t *testing.T) {
	p := NewPlugin()
	moduleProvider, ok := p.(sdk.ModuleProvider)
	if !ok {
		t.Fatal("plugin does not expose module provider methods")
	}
	if got := moduleProvider.ModuleTypes(); len(got) != 1 || got[0] != ModuleTypeAnalyticsGoogleProvider {
		t.Fatalf("ModuleTypes() = %v", got)
	}
}

func TestGoogleProviderRegistersAndUnregisters(t *testing.T) {
	module, err := newGoogleProviderModule("google", map[string]any{
		"analytics_account":   "accounts/123",
		"tag_manager_account": "accounts/456",
	})
	if err != nil {
		t.Fatalf("newGoogleProviderModule: %v", err)
	}
	if err := module.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	provider, ok := getGoogleProvider("google")
	if !ok {
		t.Fatal("provider was not registered")
	}
	if provider.config.AnalyticsAccount != "accounts/123" {
		t.Fatalf("analytics account = %q", provider.config.AnalyticsAccount)
	}
	if err := module.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if _, ok := getGoogleProvider("google"); ok {
		t.Fatal("provider remained registered after Stop")
	}
}

func TestGoogleProviderCredentialOptionsFromEnv(t *testing.T) {
	t.Setenv("GOOGLE_CREDS_JSON", `{"type":"service_account"}`)
	provider := googleProvider{config: GoogleProviderConfig{CredentialsJSONEnv: "GOOGLE_CREDS_JSON"}}
	opts, explicit, err := provider.clientOptions()
	if err != nil {
		t.Fatalf("clientOptions: %v", err)
	}
	if !explicit {
		t.Fatal("credentials JSON env should count as explicit credentials")
	}
	if len(opts) != 1 {
		t.Fatalf("client options = %d, want 1", len(opts))
	}
}

func TestGoogleProviderAllowsADCWhenExplicit(t *testing.T) {
	provider := googleProvider{config: GoogleProviderConfig{AllowADC: true}}
	opts, explicit, err := provider.clientOptions()
	if err != nil {
		t.Fatalf("clientOptions: %v", err)
	}
	if !explicit {
		t.Fatal("allow_adc should authorize live SDK construction")
	}
	if len(opts) != 0 {
		t.Fatalf("ADC should not add explicit options, got %d", len(opts))
	}
}

func TestGoogleProviderCredentialErrorRedactsValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	provider := googleProvider{config: GoogleProviderConfig{CredentialsFile: path}}
	_, _, err := provider.clientOptions()
	if err == nil {
		t.Fatal("expected missing credentials file error")
	}
	if strings.Contains(err.Error(), path) {
		t.Fatalf("credential path leaked in error: %v", err)
	}
}

func TestGoogleAuditWriterAppendsJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	writer := googleAuditWriter{path: path}
	if err := writer.Append(context.Background(), googleAuditEvent{
		Action:     "ga4.ensure",
		Account:    "accounts/123",
		Resource:   "properties/1",
		DryRun:     true,
		Operations: []string{"create_property"},
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	text := string(data)
	for _, want := range []string{`"action":"ga4.ensure"`, `"account":"accounts/123"`, `"dry_run":true`, `"create_property"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("audit missing %s: %s", want, text)
		}
	}
}
