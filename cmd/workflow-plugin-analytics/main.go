// Command workflow-plugin-analytics is a workflow engine external plugin.
// It exposes CLI tooling for injecting analytics and tag-manager snippets into
// rendered HTML assets.
package main

import (
	"github.com/GoCodeAlone/workflow-plugin-analytics/internal"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func main() {
	plugin := internal.NewPlugin()
	sdk.ServePluginFull(plugin, internal.NewCLIProvider(), nil)
}
