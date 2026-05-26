package internal

import (
	"context"
	"encoding/json"
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
	case "google":
		return c.runGoogle(args[2:])
	case "help", "-h", "--help":
		c.usage()
		return 0
	default:
		fmt.Fprintf(c.stderr, "unknown analytics subcommand %q\n", args[1])
		c.usage()
		return 2
	}
}

func (c *CLIProvider) runGoogle(args []string) int {
	if len(args) < 2 {
		c.googleUsage()
		return 2
	}
	switch args[0] + "/" + args[1] {
	case "ga4/ensure":
		return c.runGoogleGA4Ensure(args[2:])
	case "gtm/ensure":
		return c.runGoogleGTMEnsure(args[2:])
	default:
		fmt.Fprintf(c.stderr, "unknown analytics google subcommand %q\n", args)
		c.googleUsage()
		return 2
	}
}

func (c *CLIProvider) runGoogleGA4Ensure(args []string) int {
	fs := flag.NewFlagSet("analytics google ga4 ensure", flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	account := fs.String("account", "", "Google Analytics account resource, accounts/<id>")
	propertyName := fs.String("property-name", "", "GA4 property display name")
	streamName := fs.String("stream-name", "", "GA4 web stream display name")
	defaultURI := fs.String("default-uri", "", "GA4 web stream default URI")
	timeZone := fs.String("time-zone", "America/New_York", "GA4 reporting time zone")
	currency := fs.String("currency", "USD", "GA4 currency code")
	dryRun := fs.Bool("dry-run", false, "Plan changes without calling Google APIs")
	credentialsJSONEnv := fs.String("credentials-json-env", "", "Environment variable containing service-account JSON")
	credentialsFileEnv := fs.String("credentials-file-env", "", "Environment variable containing service-account JSON file path")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	provider := googleProvider{config: GoogleProviderConfig{
		CredentialsJSONEnv: *credentialsJSONEnv,
		CredentialsFileEnv: *credentialsFileEnv,
	}}
	client, audit, err := ga4ClientForProvider(context.Background(), provider, *dryRun)
	if err != nil {
		fmt.Fprintf(c.stderr, "analytics google ga4 ensure: %v\n", err)
		return 1
	}
	result, err := EnsureGA4WebStream(context.Background(), client, GA4EnsureRequest{
		Account:      *account,
		PropertyName: *propertyName,
		StreamName:   *streamName,
		DefaultURI:   *defaultURI,
		TimeZone:     *timeZone,
		CurrencyCode: *currency,
		DryRun:       *dryRun,
	})
	if err != nil {
		fmt.Fprintf(c.stderr, "analytics google ga4 ensure: %v\n", err)
		return 1
	}
	_ = audit.Append(context.Background(), googleAuditEvent{Action: "ga4.ensure", Account: result.Account, Resource: result.Property, DryRun: result.DryRun, Operations: operationNameStrings(result.Operations)})
	return c.writeJSON(result)
}

func (c *CLIProvider) runGoogleGTMEnsure(args []string) int {
	fs := flag.NewFlagSet("analytics google gtm ensure", flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	account := fs.String("account", "", "Google Tag Manager account resource, accounts/<id>")
	containerName := fs.String("container-name", "", "GTM container display name")
	workspaceName := fs.String("workspace-name", "workflow", "GTM workspace display name")
	measurementID := fs.String("measurement-id", "", "GA4 measurement ID to wire into Google tag config")
	dryRun := fs.Bool("dry-run", false, "Plan changes without calling Google APIs")
	credentialsJSONEnv := fs.String("credentials-json-env", "", "Environment variable containing service-account JSON")
	credentialsFileEnv := fs.String("credentials-file-env", "", "Environment variable containing service-account JSON file path")
	var domains repeatedFlag
	fs.Var(&domains, "domain", "Domain associated with the web container; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	provider := googleProvider{config: GoogleProviderConfig{
		CredentialsJSONEnv: *credentialsJSONEnv,
		CredentialsFileEnv: *credentialsFileEnv,
	}}
	client, audit, err := gtmClientForProvider(context.Background(), provider, *dryRun)
	if err != nil {
		fmt.Fprintf(c.stderr, "analytics google gtm ensure: %v\n", err)
		return 1
	}
	result, err := EnsureGTMWebContainer(context.Background(), client, GTMEnsureRequest{
		Account:       *account,
		ContainerName: *containerName,
		Domains:       domains,
		WorkspaceName: *workspaceName,
		MeasurementID: *measurementID,
		DryRun:        *dryRun,
	})
	if err != nil {
		fmt.Fprintf(c.stderr, "analytics google gtm ensure: %v\n", err)
		return 1
	}
	_ = audit.Append(context.Background(), googleAuditEvent{Action: "gtm.ensure", Account: result.Account, Resource: result.ContainerPath, DryRun: result.DryRun, Operations: operationNameStrings(result.Operations)})
	return c.writeJSON(result)
}

func (c *CLIProvider) writeJSON(v any) int {
	enc := json.NewEncoder(c.stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(c.stderr, "analytics: encode JSON: %v\n", err)
		return 1
	}
	return 0
}

type repeatedFlag []string

func (f *repeatedFlag) String() string { return fmt.Sprint([]string(*f)) }
func (f *repeatedFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
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
  analytics inject                 Inject provider snippets into rendered HTML assets
  analytics google ga4 ensure      Ensure GA4 property + web stream
  analytics google gtm ensure      Ensure GTM web container + workspace`)
}

func (c *CLIProvider) googleUsage() {
	fmt.Fprintln(c.stderr, `Usage:
  wfctl analytics google ga4 ensure --account accounts/123 --property-name example.com --stream-name example.com --default-uri https://example.com --dry-run
  wfctl analytics google gtm ensure --account accounts/456 --container-name example.com --domain example.com --dry-run`)
}
