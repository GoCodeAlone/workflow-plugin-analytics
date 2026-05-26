package internal

import (
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"
)

// Operation describes one reconciliation action that was reused, planned, or
// performed. It is JSON-friendly for CLI/step outputs and audit logs.
type Operation struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func planned(name string, dryRun bool) Operation {
	if dryRun {
		return Operation{Name: name, Status: "planned"}
	}
	return Operation{Name: name, Status: "created"}
}

func reused(name string) Operation {
	return Operation{Name: name, Status: "reused"}
}

func operationNameStrings(ops []Operation) []string {
	out := make([]string, 0, len(ops))
	for _, op := range ops {
		out = append(out, op.Name)
	}
	return out
}

func validateGoogleAccount(account string) error {
	if !strings.HasPrefix(account, "accounts/") || strings.TrimPrefix(account, "accounts/") == "" {
		return fmt.Errorf("account must use accounts/<id> format")
	}
	return nil
}

func validateGoogleProperty(property string) error {
	if !strings.HasPrefix(property, "properties/") || strings.TrimPrefix(property, "properties/") == "" {
		return fmt.Errorf("property must use properties/<id> format")
	}
	return nil
}

func validateWebURI(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return fmt.Errorf("default_uri must be an http(s) URL")
	}
	return nil
}

func normalizeDomains(values []string) ([]string, error) {
	seen := make(map[string]struct{}, len(values))
	var out []string
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if strings.Contains(value, "://") {
			u, err := url.Parse(value)
			if err != nil {
				return nil, fmt.Errorf("invalid domain %q", value)
			}
			value = u.Host
		}
		if strings.Contains(value, " ") || net.ParseIP(value) != nil {
			return nil, fmt.Errorf("invalid domain %q", value)
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one domain is required")
	}
	return out, nil
}

func stringSliceConfig(config map[string]any, key string) []string {
	if config == nil {
		return nil
	}
	switch v := config[key].(type) {
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			if s := strings.TrimSpace(part); s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func boolConfig(config map[string]any, key string, fallback bool) bool {
	if config == nil {
		return fallback
	}
	if v, ok := config[key].(bool); ok {
		return v
	}
	return fallback
}
