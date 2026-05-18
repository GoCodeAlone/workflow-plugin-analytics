// Package internal implements the workflow-plugin-analytics plugin.
package internal

import (
	"fmt"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

// Version is set at build time via -ldflags
// "-X github.com/GoCodeAlone/workflow-plugin-analytics/internal.Version=X.Y.Z".
// Default is a bare semver so plugin loaders that validate semver accept
// unreleased dev builds; goreleaser overrides with the real release tag.
var Version = "0.0.0"

type analyticsPlugin struct{}

// StepTypeAnalyticsInjectHTML injects analytics snippets into HTML carried by a
// Workflow pipeline, covering handlers that render HTML at runtime.
const StepTypeAnalyticsInjectHTML = "step.analytics_inject_html"

// NewPlugin returns a new plugin instance. main.go calls sdk.Serve(NewPlugin()).
func NewPlugin() sdk.PluginProvider {
	return &analyticsPlugin{}
}

// Manifest returns the plugin metadata used by the workflow engine for
// discovery and capability negotiation.
func (p *analyticsPlugin) Manifest() sdk.PluginManifest {
	return sdk.PluginManifest{
		Name:        "workflow-plugin-analytics",
		Version:     Version,
		Author:      "GoCodeAlone",
		Description: "Analytics and tag-manager injection plugin for rendered HTML assets",
	}
}

// StepTypes returns the runtime step types this plugin provides.
func (p *analyticsPlugin) StepTypes() []string {
	return []string{StepTypeAnalyticsInjectHTML}
}

// CreateStep creates an analytics step instance.
func (p *analyticsPlugin) CreateStep(typeName, name string, config map[string]any) (sdk.StepInstance, error) {
	switch typeName {
	case StepTypeAnalyticsInjectHTML:
		return newAnalyticsInjectHTMLStep(name, config)
	default:
		return nil, fmt.Errorf("analytics plugin: unknown step type %q", typeName)
	}
}
