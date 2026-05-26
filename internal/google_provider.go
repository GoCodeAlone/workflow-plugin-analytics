package internal

import (
	"context"
	"fmt"
	"os"
	"sync"

	"google.golang.org/api/option"
)

type GoogleProviderConfig struct {
	CredentialsJSON    string
	CredentialsJSONEnv string
	CredentialsFile    string
	CredentialsFileEnv string
	AnalyticsAccount   string
	TagManagerAccount  string
	AuditPath          string
	AllowADC           bool
}

type googleProvider struct {
	config GoogleProviderConfig
}

type googleProviderModule struct {
	name     string
	provider googleProvider
}

var googleProviders = struct {
	sync.RWMutex
	byName map[string]googleProvider
}{byName: make(map[string]googleProvider)}

func newGoogleProviderModule(name string, config map[string]any) (*googleProviderModule, error) {
	cfg := GoogleProviderConfig{
		CredentialsJSON:    stringValue(config, "credentials_json"),
		CredentialsJSONEnv: stringValue(config, "credentials_json_env"),
		CredentialsFile:    stringValue(config, "credentials_file"),
		CredentialsFileEnv: stringValue(config, "credentials_file_env"),
		AnalyticsAccount:   stringValue(config, "analytics_account"),
		TagManagerAccount:  stringValue(config, "tag_manager_account"),
		AuditPath:          stringValue(config, "audit_path"),
		AllowADC:           boolConfig(config, "allow_adc", false),
	}
	if cfg.AuditPath == "" {
		cfg.AuditPath = defaultGoogleAuditPath()
	}
	return &googleProviderModule{name: name, provider: googleProvider{config: cfg}}, nil
}

func (m *googleProviderModule) Init() error {
	googleProviders.Lock()
	defer googleProviders.Unlock()
	googleProviders.byName[m.name] = m.provider
	return nil
}

func (m *googleProviderModule) Start(_ context.Context) error { return nil }

func (m *googleProviderModule) Stop(_ context.Context) error {
	googleProviders.Lock()
	defer googleProviders.Unlock()
	delete(googleProviders.byName, m.name)
	return nil
}

func getGoogleProvider(name string) (googleProvider, bool) {
	if name == "" {
		name = "google"
	}
	googleProviders.RLock()
	defer googleProviders.RUnlock()
	provider, ok := googleProviders.byName[name]
	return provider, ok
}

func (p googleProvider) clientOptions() ([]option.ClientOption, bool, error) {
	if p.config.CredentialsJSON != "" {
		return []option.ClientOption{option.WithCredentialsJSON([]byte(p.config.CredentialsJSON))}, true, nil
	}
	if p.config.CredentialsJSONEnv != "" {
		value := os.Getenv(p.config.CredentialsJSONEnv)
		if value == "" {
			return nil, false, fmt.Errorf("google credentials JSON env %q is empty", p.config.CredentialsJSONEnv)
		}
		return []option.ClientOption{option.WithCredentialsJSON([]byte(value))}, true, nil
	}
	if p.config.CredentialsFile != "" {
		if _, err := os.Stat(p.config.CredentialsFile); err != nil {
			return nil, false, fmt.Errorf("google credentials file is not readable")
		}
		return []option.ClientOption{option.WithCredentialsFile(p.config.CredentialsFile)}, true, nil
	}
	if p.config.CredentialsFileEnv != "" {
		path := os.Getenv(p.config.CredentialsFileEnv)
		if path == "" {
			return nil, false, fmt.Errorf("google credentials file env %q is empty", p.config.CredentialsFileEnv)
		}
		if _, err := os.Stat(path); err != nil {
			return nil, false, fmt.Errorf("google credentials file is not readable")
		}
		return []option.ClientOption{option.WithCredentialsFile(path)}, true, nil
	}
	if p.config.AllowADC {
		return nil, true, nil
	}
	return nil, false, nil
}

func stringValue(config map[string]any, key string) string {
	v, _ := stringConfig(config, key)
	return v
}
