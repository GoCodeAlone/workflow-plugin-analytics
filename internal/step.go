package internal

import (
	"context"
	"fmt"
	"os"
	"strings"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

type analyticsInjectHTMLStep struct {
	name     string
	provider string
	tagID    string
	tagIDEnv string
	htmlKey  string
}

func newAnalyticsInjectHTMLStep(name string, config map[string]any) (*analyticsInjectHTMLStep, error) {
	step := &analyticsInjectHTMLStep{
		name:     name,
		provider: ProviderGoogleAnalytics,
		htmlKey:  "html",
	}
	if v, ok := stringConfig(config, "provider"); ok {
		step.provider = normalizeProvider(v)
	}
	if step.provider != ProviderGoogleAnalytics && step.provider != ProviderGoogleTagManager {
		return nil, fmt.Errorf("%s %q: unsupported provider %q", StepTypeAnalyticsInjectHTML, name, step.provider)
	}
	if v, ok := stringConfig(config, "tag_id"); ok {
		step.tagID = v
	}
	if v, ok := stringConfig(config, "tag_id_env"); ok {
		step.tagIDEnv = v
	}
	if v, ok := stringConfig(config, "html_field"); ok {
		step.htmlKey = v
	}
	return step, nil
}

func (s *analyticsInjectHTMLStep) Execute(
	_ context.Context,
	_ map[string]any,
	_ map[string]map[string]any,
	current map[string]any,
	_ map[string]any,
	config map[string]any,
) (*sdk.StepResult, error) {
	provider := s.provider
	if v, ok := stringConfig(config, "provider"); ok {
		provider = normalizeProvider(v)
	}
	htmlKey := s.htmlKey
	if v, ok := stringConfig(config, "html_field"); ok {
		htmlKey = v
	}
	tagID := s.tagID
	if v, ok := stringConfig(config, "tag_id"); ok {
		tagID = v
	}
	tagIDEnv := s.tagIDEnv
	if v, ok := stringConfig(config, "tag_id_env"); ok {
		tagIDEnv = v
	}
	if strings.TrimSpace(tagID) == "" && tagIDEnv != "" {
		tagID = os.Getenv(tagIDEnv)
	}

	rawHTML, ok := stringConfig(config, "html")
	if !ok && current != nil {
		rawHTML, ok = current[htmlKey].(string)
	}
	if !ok {
		return nil, fmt.Errorf("%s %q: html is required in config.html or current[%q]", StepTypeAnalyticsInjectHTML, s.name, htmlKey)
	}

	next, err := injectHTML(rawHTML, provider, strings.TrimSpace(tagID))
	if err != nil {
		return nil, fmt.Errorf("%s %q: %w", StepTypeAnalyticsInjectHTML, s.name, err)
	}
	skipped := strings.TrimSpace(tagID) == ""
	reason := ""
	if skipped {
		reason = "empty tag id"
	}
	return &sdk.StepResult{
		Output: map[string]any{
			"html":     next,
			"injected": next != rawHTML && !skipped,
			"skipped":  skipped,
			"reason":   reason,
			"provider": provider,
		},
	}, nil
}

func stringConfig(config map[string]any, key string) (string, bool) {
	if config == nil {
		return "", false
	}
	v, ok := config[key].(string)
	if !ok {
		return "", false
	}
	v = strings.TrimSpace(v)
	return v, v != ""
}
