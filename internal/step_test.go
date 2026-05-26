package internal

import (
	"context"
	"strings"
	"testing"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func TestPluginExposesAnalyticsInjectStep(t *testing.T) {
	p := NewPlugin()
	stepProvider, ok := p.(sdk.StepProvider)
	if !ok {
		t.Fatal("plugin does not expose step provider methods")
	}
	if got := stepProvider.StepTypes(); !sameStrings(got, []string{StepTypeAnalyticsInjectHTML, StepTypeAnalyticsGoogleGA4Ensure, StepTypeAnalyticsGoogleGTMEnsure}) {
		t.Fatalf("StepTypes() = %v", got)
	}
}

func TestAnalyticsInjectHTMLStepUsesRuntimeHTMLAndEnvTag(t *testing.T) {
	t.Setenv("GOOGLE_TAG_ID", "G-RUNTIME")
	step, err := newAnalyticsInjectHTMLStep("inject", map[string]any{
		"tag_id_env": "GOOGLE_TAG_ID",
	})
	if err != nil {
		t.Fatalf("newAnalyticsInjectHTMLStep: %v", err)
	}

	result, err := step.Execute(context.Background(), nil, nil, map[string]any{"html": sampleHTML}, nil, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	html, ok := result.Output["html"].(string)
	if !ok {
		t.Fatalf("html output missing: %#v", result.Output)
	}
	assertContains(t, html, "G-RUNTIME")
	if result.Output["injected"] != true || result.Output["skipped"] != false {
		t.Fatalf("unexpected flags: %#v", result.Output)
	}
}

func TestAnalyticsInjectHTMLStepEmptyEnvNoops(t *testing.T) {
	t.Setenv("GOOGLE_TAG_ID", "")
	step, err := newAnalyticsInjectHTMLStep("inject", map[string]any{
		"tag_id_env": "GOOGLE_TAG_ID",
	})
	if err != nil {
		t.Fatalf("newAnalyticsInjectHTMLStep: %v", err)
	}

	result, err := step.Execute(context.Background(), nil, nil, map[string]any{"html": sampleHTML}, nil, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	html := result.Output["html"].(string)
	if html != sampleHTML {
		t.Fatalf("empty env should leave unmanaged HTML unchanged:\n%s", html)
	}
	if strings.Contains(html, "googletagmanager.com") {
		t.Fatalf("empty env injected tag:\n%s", html)
	}
	if result.Output["injected"] != false || result.Output["skipped"] != true {
		t.Fatalf("unexpected flags: %#v", result.Output)
	}
}

func TestAnalyticsInjectHTMLStepAnonymizeIP(t *testing.T) {
	step, err := newAnalyticsInjectHTMLStep("inj", map[string]any{
		"tag_id":       "G-TENANTA",
		"anonymize_ip": true,
	})
	if err != nil {
		t.Fatalf("new step: %v", err)
	}
	res, err := step.Execute(
		context.Background(), nil, nil,
		map[string]any{"html": "<html><head></head><body></body></html>"},
		nil, nil,
	)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	out, _ := res.Output["html"].(string)
	if !strings.Contains(out, "'anonymize_ip': true") {
		t.Errorf("step did not honour anonymize_ip flag; got %q", out)
	}
	if !strings.Contains(out, "G-TENANTA") {
		t.Errorf("step missing tenant tag_id; got %q", out)
	}
}

func TestAnalyticsInjectHTMLStepPerCallTagID(t *testing.T) {
	// Multi-tenant invocation: tag_id passed at execute time (e.g. from
	// tenant-resolved context) rather than configured at module build.
	step, err := newAnalyticsInjectHTMLStep("inj", map[string]any{})
	if err != nil {
		t.Fatalf("new step: %v", err)
	}
	res, err := step.Execute(
		context.Background(), nil, nil,
		map[string]any{"html": "<html><head></head><body></body></html>"},
		nil,
		map[string]any{"tag_id": "G-TENANTB", "anonymize_ip": true},
	)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	out, _ := res.Output["html"].(string)
	if !strings.Contains(out, "G-TENANTB") {
		t.Errorf("per-call tag_id not honoured; got %q", out)
	}
	if !strings.Contains(out, "'anonymize_ip': true") {
		t.Errorf("per-call anonymize_ip not honoured; got %q", out)
	}
}

func TestAnalyticsGoogleGA4EnsureStepDryRun(t *testing.T) {
	step, err := newAnalyticsGoogleGA4EnsureStep("ga4", map[string]any{
		"account":       "accounts/123",
		"property_name": "example.com",
		"stream_name":   "example.com",
		"default_uri":   "https://example.com",
		"dry_run":       true,
	})
	if err != nil {
		t.Fatalf("new step: %v", err)
	}
	res, err := step.Execute(context.Background(), nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Output["dry_run"] != true {
		t.Fatalf("dry_run output = %#v", res.Output)
	}
	if res.Output["measurement_id"] != "" {
		t.Fatalf("measurement_id = %#v", res.Output["measurement_id"])
	}
}

func TestAnalyticsGoogleGA4EnsureStepAcceptsExplicitProperty(t *testing.T) {
	step, err := newAnalyticsGoogleGA4EnsureStep("ga4", map[string]any{
		"account":     "accounts/395146029",
		"property":    "properties/538139248",
		"stream_name": "gocodealone.tech",
		"default_uri": "https://gocodealone.tech",
		"dry_run":     true,
	})
	if err != nil {
		t.Fatalf("new step: %v", err)
	}
	res, err := step.Execute(context.Background(), nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Output["property"] != "properties/538139248" {
		t.Fatalf("property output = %#v", res.Output)
	}
}

func TestAnalyticsGoogleGTMEnsureStepDryRun(t *testing.T) {
	step, err := newAnalyticsGoogleGTMEnsureStep("gtm", map[string]any{
		"account":        "accounts/456",
		"container_name": "example.com",
		"domains":        []any{"example.com"},
		"workspace_name": "workflow",
		"measurement_id": "G-ABC123",
		"dry_run":        true,
	})
	if err != nil {
		t.Fatalf("new step: %v", err)
	}
	res, err := step.Execute(context.Background(), nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Output["dry_run"] != true {
		t.Fatalf("dry_run output = %#v", res.Output)
	}
	if res.Output["public_id"] != "" {
		t.Fatalf("public_id = %#v", res.Output["public_id"])
	}
}
