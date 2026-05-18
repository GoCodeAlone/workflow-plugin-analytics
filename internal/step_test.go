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
	if got := stepProvider.StepTypes(); len(got) != 1 || got[0] != StepTypeAnalyticsInjectHTML {
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
