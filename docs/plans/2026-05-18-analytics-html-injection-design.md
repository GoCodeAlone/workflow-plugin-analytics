# Analytics HTML Injection Design

## Goal

Ship public Workflow plugin that injects analytics/tag-manager snippets into rendered HTML when env-specific tag id exists.

## Approaches

1. Build-time CLI only.
   - Pros: simple, fits BMW static UI build.
   - Cons: misses Workflow handlers that render HTML directly.
2. Runtime middleware only.
   - Pros: central injection for every response.
   - Cons: risks mutating non-HTML responses; no existing generic public middleware contract.
3. CLI + runtime step.
   - Pros: covers static assets and handler-rendered HTML; same pure injector logic; explicit per-flow wiring.
   - Cons: apps must opt handler flows into `step.analytics_inject_html`.

Chosen: option 3.

## Design

- Provider support:
  - `google-analytics` injects GA4 `gtag.js` block into `<head>`.
  - `google-tag-manager` injects GTM script into `<head>` and noscript iframe after `<body>`.
- CLI:
  - `wfctl analytics inject --provider google-analytics --tag-id-env GOOGLE_TAG_ID --dir dist`
  - Empty tag id → remove managed block, skip injection.
- Runtime:
  - `step.analytics_inject_html`.
  - Inputs: `provider`, `tag_id`, `tag_id_env`, `html`, `html_field`.
  - Reads HTML from `config.html` or `current[html_field]`.
  - Outputs: `html`, `injected`, `skipped`, `reason`, `provider`.
- Double injection guard:
  - Remove plugin-managed blocks before reinjection.
  - If unmanaged snippet with same provider id exists, leave HTML unchanged.

## Assumptions

- Google tag IDs and GTM container IDs fit `[A-Za-z0-9_-]+`.
- BMW prod build can read `GOOGLE_TAG_ID` from GitHub `prod` environment.
- Workflow handler HTML injection should be explicit step wiring, not automatic response mutation.
- Existing unmanaged snippets with different IDs are not removed.

## Self-Challenge

- Could BMW use `sed` in deploy only? Yes, but then no reusable Workflow primitive and no handler-rendered path.
- Fragile assumption: one image for staging+prod cannot carry prod-only env-specific GA; BMW deploy must build env-specific images.
- YAGNI risk: GTM now. Kept because plugin purpose is tag-manager-capable and GTM uses same injection core.

## Plan

1. Add pure injector with managed-block replacement, same-ID unmanaged detection, empty-ID skip.
2. Expose `wfctl analytics inject` CLI.
3. Expose `step.analytics_inject_html` for handler-rendered HTML.
4. Add unit tests for idempotency, skip, unmanaged double-injection guard, runtime step.
5. Validate plugin manifest and publish public release.
6. Wire BMW deploy to env-specific analytics injection.

## Verification

- `GOWORK=off go test ./... -race -count=1`
- `GOWORK=off go vet ./...`
- `GOWORK=off go build ./...`
- `wfctl plugin validate --file plugin.json --strict-contracts`
- BMW deploy smoke before prod.

## Rollback

- Revert BMW plugin pin and deploy injection step.
- Redeploy previous BMW image.
- Remove analytics plugin from registry manifest if release broken.
