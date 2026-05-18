package internal

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	ProviderGoogleAnalytics  = "google-analytics"
	ProviderGoogleTagManager = "google-tag-manager"
)

var safeTagIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// InjectOptions controls HTML tag injection.
type InjectOptions struct {
	Provider string
	TagID    string
	HTMLPath string
	Dir      string
	DryRun   bool
}

// InjectSummary reports what an injection run did.
type InjectSummary struct {
	FilesChanged int
	FilesChecked int
	Skipped      bool
	Reason       string
}

// Inject applies the selected provider snippet to one HTML file or a directory
// tree. Empty TagID is a no-op that removes any existing managed block.
func Inject(opts InjectOptions) (InjectSummary, error) {
	provider := normalizeProvider(opts.Provider)
	if provider == "" {
		return InjectSummary{}, fmt.Errorf("provider is required")
	}
	if provider != ProviderGoogleAnalytics && provider != ProviderGoogleTagManager {
		return InjectSummary{}, fmt.Errorf("unsupported provider %q", opts.Provider)
	}
	if opts.HTMLPath == "" && opts.Dir == "" {
		return InjectSummary{}, fmt.Errorf("one of --html or --dir is required")
	}
	if opts.HTMLPath != "" && opts.Dir != "" {
		return InjectSummary{}, fmt.Errorf("--html and --dir are mutually exclusive")
	}
	tagID := strings.TrimSpace(opts.TagID)
	if tagID != "" && !safeTagIDPattern.MatchString(tagID) {
		return InjectSummary{}, fmt.Errorf("tag id contains unsupported characters")
	}

	paths, err := htmlPaths(opts.HTMLPath, opts.Dir)
	if err != nil {
		return InjectSummary{}, err
	}
	summary := InjectSummary{FilesChecked: len(paths)}
	if tagID == "" {
		summary.Skipped = true
		summary.Reason = "empty tag id"
	}
	for _, path := range paths {
		changed, err := injectFile(path, provider, tagID, opts.DryRun)
		if err != nil {
			return summary, err
		}
		if changed {
			summary.FilesChanged++
		}
	}
	return summary, nil
}

func normalizeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "ga", "ga4", "google_analytics", "google-analytics":
		return ProviderGoogleAnalytics
	case "gtm", "google_tag_manager", "google-tag-manager":
		return ProviderGoogleTagManager
	default:
		return strings.ToLower(strings.TrimSpace(provider))
	}
}

func htmlPaths(htmlPath, dir string) ([]string, error) {
	if htmlPath != "" {
		return []string{htmlPath}, nil
	}
	var paths []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".html") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("no .html files found under %s", dir)
	}
	return paths, nil
}

func injectFile(path, provider, tagID string, dryRun bool) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read %s: %w", path, err)
	}
	original := string(data)
	next, err := injectHTML(original, provider, tagID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}
	if next == original {
		return false, nil
	}
	if dryRun {
		return true, nil
	}
	if err := os.WriteFile(path, []byte(next), 0644); err != nil {
		return false, fmt.Errorf("write %s: %w", path, err)
	}
	return true, nil
}

func injectHTML(input, provider, tagID string) (string, error) {
	htmlDoc := removeManagedBlocks(input, provider)
	if tagID == "" {
		return htmlDoc, nil
	}
	if providerSnippetPresent(htmlDoc, provider, tagID) {
		return htmlDoc, nil
	}
	switch provider {
	case ProviderGoogleAnalytics:
		return injectBeforeClosingTag(htmlDoc, "</head>", googleAnalyticsBlock(tagID))
	case ProviderGoogleTagManager:
		withHead, err := injectBeforeClosingTag(htmlDoc, "</head>", googleTagManagerHeadBlock(tagID))
		if err != nil {
			return "", err
		}
		return injectAfterOpeningBody(withHead, googleTagManagerBodyBlock(tagID))
	default:
		return "", fmt.Errorf("unsupported provider %q", provider)
	}
}

func providerSnippetPresent(input, provider, tagID string) bool {
	idURL := url.QueryEscape(tagID)
	switch provider {
	case ProviderGoogleAnalytics:
		return strings.Contains(input, "googletagmanager.com/gtag/js?id="+idURL) ||
			strings.Contains(input, "gtag('config', '"+tagID+"'") ||
			strings.Contains(input, `gtag("config", "`+tagID+`"`)
	case ProviderGoogleTagManager:
		return strings.Contains(input, "googletagmanager.com/gtm.js?id="+idURL) ||
			strings.Contains(input, "googletagmanager.com/ns.html?id="+idURL) ||
			strings.Contains(input, ",'"+tagID+"'") ||
			strings.Contains(input, `,"`+tagID+`"`)
	default:
		return false
	}
}

func removeManagedBlocks(input, provider string) string {
	out := removeBlock(input, blockStart(provider, "head"), blockEnd(provider, "head"))
	return removeBlock(out, blockStart(provider, "body"), blockEnd(provider, "body"))
}

func removeBlock(input, start, end string) string {
	for {
		startIdx := strings.Index(input, start)
		if startIdx < 0 {
			return input
		}
		endIdx := strings.Index(input[startIdx:], end)
		if endIdx < 0 {
			return input
		}
		endIdx += startIdx + len(end)
		for endIdx < len(input) && (input[endIdx] == '\n' || input[endIdx] == '\r') {
			endIdx++
		}
		input = input[:startIdx] + input[endIdx:]
	}
}

func injectBeforeClosingTag(input, tag, block string) (string, error) {
	idx := strings.LastIndex(strings.ToLower(input), strings.ToLower(tag))
	if idx < 0 {
		return "", fmt.Errorf("missing %s", tag)
	}
	return input[:idx] + block + input[idx:], nil
}

func injectAfterOpeningBody(input, block string) (string, error) {
	lower := strings.ToLower(input)
	bodyIdx := strings.Index(lower, "<body")
	if bodyIdx < 0 {
		return "", errors.New("missing <body>")
	}
	closeIdx := strings.Index(lower[bodyIdx:], ">")
	if closeIdx < 0 {
		return "", errors.New("malformed <body> tag")
	}
	insertAt := bodyIdx + closeIdx + 1
	return input[:insertAt] + "\n" + block + input[insertAt:], nil
}

func blockStart(provider, slot string) string {
	return fmt.Sprintf("<!-- workflow-plugin-analytics:%s:%s:start -->", provider, slot)
}

func blockEnd(provider, slot string) string {
	return fmt.Sprintf("<!-- workflow-plugin-analytics:%s:%s:end -->", provider, slot)
}

func googleAnalyticsBlock(tagID string) string {
	idJS := escapeJSString(tagID)
	idURL := url.QueryEscape(tagID)
	return fmt.Sprintf(`%s
<script async src="https://www.googletagmanager.com/gtag/js?id=%s"></script>
<script>
  window.dataLayer = window.dataLayer || [];
  function gtag(){dataLayer.push(arguments);}
  gtag('js', new Date());
  gtag('config', '%s');
</script>
%s
`, blockStart(ProviderGoogleAnalytics, "head"), idURL, idJS, blockEnd(ProviderGoogleAnalytics, "head"))
}

func googleTagManagerHeadBlock(tagID string) string {
	idJS := escapeJSString(tagID)
	return fmt.Sprintf(`%s
<script>
  (function(w,d,s,l,i){w[l]=w[l]||[];w[l].push({'gtm.start':
  new Date().getTime(),event:'gtm.js'});var f=d.getElementsByTagName(s)[0],
  j=d.createElement(s),dl=l!='dataLayer'?'&l='+l:'';j.async=true;j.src=
  'https://www.googletagmanager.com/gtm.js?id='+i+dl;f.parentNode.insertBefore(j,f);
  })(window,document,'script','dataLayer','%s');
</script>
%s
`, blockStart(ProviderGoogleTagManager, "head"), idJS, blockEnd(ProviderGoogleTagManager, "head"))
}

func googleTagManagerBodyBlock(tagID string) string {
	idURL := url.QueryEscape(tagID)
	return fmt.Sprintf(`%s
<noscript><iframe src="https://www.googletagmanager.com/ns.html?id=%s"
height="0" width="0" style="display:none;visibility:hidden"></iframe></noscript>
%s
`, blockStart(ProviderGoogleTagManager, "body"), idURL, blockEnd(ProviderGoogleTagManager, "body"))
}

func escapeJSString(s string) string {
	return strings.ReplaceAll(s, `'`, `\'`)
}
