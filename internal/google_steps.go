package internal

import (
	"context"
	"fmt"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

type analyticsGoogleGA4EnsureStep struct {
	name   string
	config map[string]any
}

type analyticsGoogleGTMEnsureStep struct {
	name   string
	config map[string]any
}

func newAnalyticsGoogleGA4EnsureStep(name string, config map[string]any) (*analyticsGoogleGA4EnsureStep, error) {
	return &analyticsGoogleGA4EnsureStep{name: name, config: copyConfig(config)}, nil
}

func newAnalyticsGoogleGTMEnsureStep(name string, config map[string]any) (*analyticsGoogleGTMEnsureStep, error) {
	return &analyticsGoogleGTMEnsureStep{name: name, config: copyConfig(config)}, nil
}

func (s *analyticsGoogleGA4EnsureStep) Execute(ctx context.Context, _ map[string]any, _ map[string]map[string]any, current map[string]any, _ map[string]any, config map[string]any) (*sdk.StepResult, error) {
	merged := mergeConfig(s.config, config)
	req := GA4EnsureRequest{
		Account:      stringValue(merged, "account"),
		PropertyName: stringValue(merged, "property_name"),
		StreamName:   stringValue(merged, "stream_name"),
		DefaultURI:   stringValue(merged, "default_uri"),
		TimeZone:     stringValue(merged, "time_zone"),
		CurrencyCode: stringValue(merged, "currency_code"),
		DryRun:       boolConfig(merged, "dry_run", false),
	}
	if req.PropertyName == "" && current != nil {
		req.PropertyName, _ = current["property_name"].(string)
	}
	provider, _ := providerFromConfig(merged)
	if req.Account == "" {
		req.Account = provider.config.AnalyticsAccount
	}
	client, audit, err := ga4ClientForProvider(ctx, provider, req.DryRun)
	if err != nil {
		return nil, fmt.Errorf("%s %q: %w", StepTypeAnalyticsGoogleGA4Ensure, s.name, err)
	}
	result, err := EnsureGA4WebStream(ctx, client, req)
	if err != nil {
		return nil, fmt.Errorf("%s %q: %w", StepTypeAnalyticsGoogleGA4Ensure, s.name, err)
	}
	_ = audit.Append(ctx, googleAuditEvent{Action: "ga4.ensure", Account: result.Account, Resource: result.Property, DryRun: result.DryRun, Operations: operationNameStrings(result.Operations)})
	return &sdk.StepResult{Output: map[string]any{
		"account":        result.Account,
		"property":       result.Property,
		"data_stream":    result.DataStream,
		"measurement_id": result.MeasurementID,
		"dry_run":        result.DryRun,
		"operations":     operationsOutput(result.Operations),
	}}, nil
}

func (s *analyticsGoogleGTMEnsureStep) Execute(ctx context.Context, _ map[string]any, _ map[string]map[string]any, _ map[string]any, _ map[string]any, config map[string]any) (*sdk.StepResult, error) {
	merged := mergeConfig(s.config, config)
	req := GTMEnsureRequest{
		Account:       stringValue(merged, "account"),
		ContainerName: stringValue(merged, "container_name"),
		Domains:       stringSliceConfig(merged, "domains"),
		WorkspaceName: stringValue(merged, "workspace_name"),
		MeasurementID: stringValue(merged, "measurement_id"),
		DryRun:        boolConfig(merged, "dry_run", false),
	}
	provider, _ := providerFromConfig(merged)
	if req.Account == "" {
		req.Account = provider.config.TagManagerAccount
	}
	client, audit, err := gtmClientForProvider(ctx, provider, req.DryRun)
	if err != nil {
		return nil, fmt.Errorf("%s %q: %w", StepTypeAnalyticsGoogleGTMEnsure, s.name, err)
	}
	result, err := EnsureGTMWebContainer(ctx, client, req)
	if err != nil {
		return nil, fmt.Errorf("%s %q: %w", StepTypeAnalyticsGoogleGTMEnsure, s.name, err)
	}
	_ = audit.Append(ctx, googleAuditEvent{Action: "gtm.ensure", Account: result.Account, Resource: result.ContainerPath, DryRun: result.DryRun, Operations: operationNameStrings(result.Operations)})
	return &sdk.StepResult{Output: map[string]any{
		"account":          result.Account,
		"container_path":   result.ContainerPath,
		"container_id":     result.ContainerID,
		"public_id":        result.PublicID,
		"workspace_path":   result.WorkspacePath,
		"gtag_config_path": result.GtagConfigPath,
		"dry_run":          result.DryRun,
		"operations":       operationsOutput(result.Operations),
	}}, nil
}

func ga4ClientForProvider(ctx context.Context, provider googleProvider, dryRun bool) (GA4AdminClient, googleAuditWriter, error) {
	audit := googleAuditWriter{path: provider.config.AuditPath}
	opts, explicit, err := provider.clientOptions()
	if err != nil {
		return nil, audit, err
	}
	if dryRun {
		return nil, audit, nil
	}
	if !explicit {
		return nil, audit, fmt.Errorf("google credentials are required for live GA4 ensure")
	}
	client, err := newGoogleGA4SDKClient(ctx, opts...)
	return client, audit, err
}

func gtmClientForProvider(ctx context.Context, provider googleProvider, dryRun bool) (TagManagerClient, googleAuditWriter, error) {
	audit := googleAuditWriter{path: provider.config.AuditPath}
	opts, explicit, err := provider.clientOptions()
	if err != nil {
		return nil, audit, err
	}
	if dryRun {
		return nil, audit, nil
	}
	if !explicit {
		return nil, audit, fmt.Errorf("google credentials are required for live GTM ensure")
	}
	client, err := newGoogleTagManagerSDKClient(ctx, opts...)
	return client, audit, err
}

func providerFromConfig(config map[string]any) (googleProvider, bool) {
	name := stringValue(config, "provider")
	if name == "" {
		name = stringValue(config, "module")
	}
	if provider, ok := getGoogleProvider(name); ok {
		return provider, true
	}
	return googleProvider{config: GoogleProviderConfig{
		CredentialsJSON:    stringValue(config, "credentials_json"),
		CredentialsJSONEnv: stringValue(config, "credentials_json_env"),
		CredentialsFile:    stringValue(config, "credentials_file"),
		CredentialsFileEnv: stringValue(config, "credentials_file_env"),
		AuditPath:          stringValue(config, "audit_path"),
	}}, false
}

func copyConfig(config map[string]any) map[string]any {
	out := make(map[string]any, len(config))
	for k, v := range config {
		out[k] = v
	}
	return out
}

func mergeConfig(base, override map[string]any) map[string]any {
	out := copyConfig(base)
	for k, v := range override {
		out[k] = v
	}
	return out
}

func operationsOutput(ops []Operation) []map[string]any {
	out := make([]map[string]any, 0, len(ops))
	for _, op := range ops {
		out = append(out, map[string]any{"name": op.Name, "status": op.Status})
	}
	return out
}
