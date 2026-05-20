package internal

import (
	"flag"
	"fmt"
	"io"
	"os"
)

type CLIProvider struct {
	stdout io.Writer
	stderr io.Writer
}

func NewCLIProvider() *CLIProvider {
	return &CLIProvider{stdout: os.Stdout, stderr: os.Stderr}
}

func newCLIProvider(stdout, stderr io.Writer) *CLIProvider {
	return &CLIProvider{stdout: stdout, stderr: stderr}
}

func (c *CLIProvider) RunCLI(args []string) int {
	if len(args) == 0 {
		c.usage()
		return 2
	}
	if args[0] != "analytics" {
		fmt.Fprintf(c.stderr, "unknown command %q\n", args[0])
		c.usage()
		return 2
	}
	if len(args) < 2 {
		c.usage()
		return 2
	}
	switch args[1] {
	case "inject":
		return c.runInject(args[2:])
	case "help", "-h", "--help":
		c.usage()
		return 0
	default:
		fmt.Fprintf(c.stderr, "unknown analytics subcommand %q\n", args[1])
		c.usage()
		return 2
	}
}

func (c *CLIProvider) runInject(args []string) int {
	fs := flag.NewFlagSet("analytics inject", flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	provider := fs.String("provider", ProviderGoogleAnalytics, "Provider: google-analytics or google-tag-manager")
	tagID := fs.String("tag-id", "", "Provider tag/container ID")
	tagIDEnv := fs.String("tag-id-env", "", "Environment variable containing the provider tag/container ID")
	htmlPath := fs.String("html", "", "HTML file to mutate")
	dir := fs.String("dir", "", "Directory to recursively process for .html files")
	dryRun := fs.Bool("dry-run", false, "Report changes without writing files")
	anonymizeIP := fs.Bool("anonymize-ip", false, "Emit anonymize_ip:true in the GA4 config (privacy mode)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	id := *tagID
	if id == "" && *tagIDEnv != "" {
		id = os.Getenv(*tagIDEnv)
	}
	summary, err := Inject(InjectOptions{
		Provider:    *provider,
		TagID:       id,
		HTMLPath:    *htmlPath,
		Dir:         *dir,
		DryRun:      *dryRun,
		AnonymizeIP: *anonymizeIP,
	})
	if err != nil {
		fmt.Fprintf(c.stderr, "analytics inject: %v\n", err)
		return 1
	}
	if summary.Skipped {
		fmt.Fprintf(c.stdout, "analytics inject: skipped (%s); checked %d HTML file(s), changed %d\n", summary.Reason, summary.FilesChecked, summary.FilesChanged)
		return 0
	}
	fmt.Fprintf(c.stdout, "analytics inject: checked %d HTML file(s), changed %d\n", summary.FilesChecked, summary.FilesChanged)
	return 0
}

func (c *CLIProvider) usage() {
	fmt.Fprintln(c.stderr, `Usage:
  wfctl analytics inject --provider google-analytics --tag-id-env GOOGLE_TAG_ID --dir dist

Subcommands:
  analytics inject   Inject provider snippets into rendered HTML assets`)
}
