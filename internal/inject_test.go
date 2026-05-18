package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleHTML = `<!doctype html>
<html>
<head>
  <title>Test</title>
</head>
<body>
  <div id="root"></div>
</body>
</html>
`

func TestInjectHTML_GoogleAnalytics(t *testing.T) {
	got, err := injectHTML(sampleHTML, ProviderGoogleAnalytics, "G-ABC123")
	if err != nil {
		t.Fatalf("injectHTML: %v", err)
	}
	assertContains(t, got, `https://www.googletagmanager.com/gtag/js?id=G-ABC123`)
	assertContains(t, got, `gtag('config', 'G-ABC123');`)
	if strings.Count(got, "workflow-plugin-analytics:google-analytics:head:start") != 1 {
		t.Fatalf("expected one managed GA block, got:\n%s", got)
	}
}

func TestInjectHTML_GoogleAnalyticsIdempotent(t *testing.T) {
	first, err := injectHTML(sampleHTML, ProviderGoogleAnalytics, "G-FIRST")
	if err != nil {
		t.Fatalf("first inject: %v", err)
	}
	second, err := injectHTML(first, ProviderGoogleAnalytics, "G-SECOND")
	if err != nil {
		t.Fatalf("second inject: %v", err)
	}
	if strings.Contains(second, "G-FIRST") {
		t.Fatalf("old tag ID remained after reinject:\n%s", second)
	}
	if strings.Count(second, "workflow-plugin-analytics:google-analytics:head:start") != 1 {
		t.Fatalf("expected one managed GA block, got:\n%s", second)
	}
	assertContains(t, second, "G-SECOND")
}

func TestInjectHTML_GoogleAnalyticsSkipsUnmanagedSameTag(t *testing.T) {
	existing := strings.Replace(sampleHTML, "</head>", `<script async src="https://www.googletagmanager.com/gtag/js?id=G-EXISTING"></script>
<script>gtag('config', 'G-EXISTING');</script>
</head>`, 1)

	got, err := injectHTML(existing, ProviderGoogleAnalytics, "G-EXISTING")
	if err != nil {
		t.Fatalf("injectHTML: %v", err)
	}
	if got != existing {
		t.Fatalf("expected unmanaged same-tag GA snippet to remain untouched:\n%s", got)
	}
	if strings.Contains(got, "workflow-plugin-analytics:google-analytics") {
		t.Fatalf("managed block added despite existing unmanaged GA snippet:\n%s", got)
	}
}

func TestInjectHTML_EmptyTagIDRemovesManagedBlock(t *testing.T) {
	withTag, err := injectHTML(sampleHTML, ProviderGoogleAnalytics, "G-ABC123")
	if err != nil {
		t.Fatalf("inject: %v", err)
	}
	withoutTag, err := injectHTML(withTag, ProviderGoogleAnalytics, "")
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if strings.Contains(withoutTag, "googletagmanager.com") {
		t.Fatalf("managed tag remained after empty ID:\n%s", withoutTag)
	}
}

func TestInjectHTML_GoogleTagManager(t *testing.T) {
	got, err := injectHTML(sampleHTML, ProviderGoogleTagManager, "GTM-ABC123")
	if err != nil {
		t.Fatalf("injectHTML: %v", err)
	}
	assertContains(t, got, `googletagmanager.com/gtm.js`)
	assertContains(t, got, `googletagmanager.com/ns.html?id=GTM-ABC123`)
	headIdx := strings.Index(got, "workflow-plugin-analytics:google-tag-manager:head:start")
	bodyIdx := strings.Index(got, "workflow-plugin-analytics:google-tag-manager:body:start")
	if headIdx < 0 || bodyIdx < 0 || headIdx > bodyIdx {
		t.Fatalf("expected GTM head block before body block, got:\n%s", got)
	}
}

func TestInjectHTML_GoogleTagManagerSkipsUnmanagedSameTag(t *testing.T) {
	existing := strings.Replace(sampleHTML, "</head>", `<script src="https://www.googletagmanager.com/gtm.js?id=GTM-EXISTING"></script>
</head>`, 1)

	got, err := injectHTML(existing, ProviderGoogleTagManager, "GTM-EXISTING")
	if err != nil {
		t.Fatalf("injectHTML: %v", err)
	}
	if got != existing {
		t.Fatalf("expected unmanaged same-tag GTM snippet to remain untouched:\n%s", got)
	}
	if strings.Contains(got, "workflow-plugin-analytics:google-tag-manager") {
		t.Fatalf("managed block added despite existing unmanaged GTM snippet:\n%s", got)
	}
}

func TestInject_DirProcessesHTMLOnly(t *testing.T) {
	dir := t.TempDir()
	htmlPath := filepath.Join(dir, "index.html")
	txtPath := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(htmlPath, []byte(sampleHTML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(txtPath, []byte(sampleHTML), 0644); err != nil {
		t.Fatal(err)
	}
	summary, err := Inject(InjectOptions{
		Provider: ProviderGoogleAnalytics,
		TagID:    "G-ABC123",
		Dir:      dir,
	})
	if err != nil {
		t.Fatalf("Inject: %v", err)
	}
	if summary.FilesChecked != 1 || summary.FilesChanged != 1 {
		t.Fatalf("summary = %+v", summary)
	}
	txt, err := os.ReadFile(txtPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(txt) != sampleHTML {
		t.Fatalf("non-HTML file changed")
	}
}

func TestInject_RejectsUnsafeTagID(t *testing.T) {
	_, err := Inject(InjectOptions{
		Provider: ProviderGoogleAnalytics,
		TagID:    `G-ABC"></script>`,
		HTMLPath: "index.html",
	})
	if err == nil {
		t.Fatal("expected unsafe tag id error")
	}
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("missing %q in:\n%s", want, got)
	}
}
