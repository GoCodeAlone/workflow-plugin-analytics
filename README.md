# Workflow Analytics Plugin

> ✅ **Verified** — used in production at **buymywishlist**. This plugin has been validated end-to-end in a merged main-branch wfctl.yaml of an active GoCodeAlone project.

`workflow-plugin-analytics` injects analytics and tag-manager snippets into rendered HTML assets from `wfctl`. It can also provision Google Analytics 4 web streams and Google Tag Manager web containers programmatically.

The first supported provider is Google Analytics through the Google tag (`gtag.js`). The plugin also includes Google Tag Manager snippet injection so apps can switch to a container-managed setup later.

## CLI

```sh
wfctl analytics inject \
  --provider google-analytics \
  --tag-id-env GOOGLE_TAG_ID \
  --dir ui/dist
```

If the selected tag ID is empty, injection is skipped and any existing managed block is removed. This lets production enable analytics with an environment variable while staging and local builds stay untracked.

## Runtime Step

Apps that render HTML directly from Workflow handlers can run the same injector as a pipeline step:

```yaml
- name: inject_analytics
  type: step.analytics_inject_html
  config:
    provider: google-analytics
    tag_id_env: GOOGLE_TAG_ID
    html_field: html
```

The step reads HTML from `config.html` or `current[html_field]`, writes the mutated document to output key `html`, and returns `injected`, `skipped`, `reason`, and `provider` metadata.

## Per-tenant (multi-app) usage

The step accepts `tag_id` and `anonymize_ip` at execute time as well as at
build time. A multi-tenant host (e.g. `gocodealone-multisite`) resolves the
tenant first, then passes per-tenant values via the step's runtime config:

```yaml
- name: inject_analytics
  type: step.analytics_inject_html
  config:
    provider: google-analytics
    # tag_id + anonymize_ip omitted here — resolved per request from
    # the tenant's multisite.yaml and merged into config at dispatch.
    html_field: html
```

If the tenant has no `analytics.google.measurement_id`, the step short-
circuits to `skipped: true, reason: "empty tag id"`. When the tenant sets
`anonymize_ip: true`, the GA4 `config(...)` call emits `{'anonymize_ip':
true}`.

## Google provisioning in wfctl YAML

The primary provisioning surface is Workflow/wfctl YAML. Put the desired GA/GTM resources in the same config that owns deploy and secret wiring, usually `deploy.yaml` or `infra.yaml`.

```yaml
secretStores:
  github-actions:
    provider: github
    config:
      repo: GoCodeAlone/example-site
      token_env: RELEASES_TOKEN

secrets:
  defaultStore: github-actions
  entries:
    - name: GOOGLE_ANALYTICS_ADMIN_CREDENTIALS_JSON
      description: Service-account JSON with GA Admin + GTM access.

modules:
  - name: google-analytics
    type: analytics.google_provider
    config:
      credentials_json: ${GOOGLE_ANALYTICS_ADMIN_CREDENTIALS_JSON}
      # Set allow_adc: true only for runners where Application Default
      # Credentials are the intended deploy identity.
      allow_adc: false
      analytics_account: accounts/123456789
      tag_manager_account: accounts/987654321

pipelines:
  apply:
    steps:
      - name: ensure_ga4
        type: step.analytics_google_ga4_ensure
        config:
          provider: google-analytics
          property_name: example.com
          stream_name: example.com
          default_uri: https://example.com
          dry_run: true

      - name: ensure_gtm
        type: step.analytics_google_gtm_ensure
        config:
          provider: google-analytics
          container_name: example.com
          domains: [example.com, www.example.com]
          workspace_name: workflow
          measurement_id: ${ensure_ga4.measurement_id}
          dry_run: true
```

`wfctl infra apply -c deploy.yaml` can run the `pipelines.apply` path for config-driven resources. Keep `dry_run: true` until Google API access and account permissions are in place.

The CLI is still useful for smoke checks and one-off operator probes. Dry-run GA4 provisioning:

```sh
wfctl analytics google ga4 ensure \
  --account accounts/123456789 \
  --property-name example.com \
  --stream-name example.com \
  --default-uri https://example.com \
  --credentials-json-env GOOGLE_ANALYTICS_ADMIN_CREDENTIALS_JSON \
  --dry-run
```

Dry-run GTM provisioning:

```sh
wfctl analytics google gtm ensure \
  --account accounts/987654321 \
  --container-name example.com \
  --domain example.com \
  --measurement-id G-XXXXXXXXXX \
  --credentials-json-env GOOGLE_ANALYTICS_ADMIN_CREDENTIALS_JSON \
  --dry-run
```

Live apply requires Google API access and credentials. Audit events for state-mutating and dry-run ensure commands are appended to `${XDG_STATE_HOME:-$HOME/.local/state}/wfctl/plugins/analytics/google-audit.jsonl` unless `audit_path` overrides it.

## Providers

- `google-analytics`: injects the Google tag into `<head>`.
- `google-tag-manager`: injects the GTM script into `<head>` and the noscript iframe immediately after `<body>`.

## Safety

- Only IDs containing letters, numbers, `_`, and `-` are accepted.
- Managed blocks are replaced idempotently.
- Existing unmanaged snippets for the same provider ID are detected and left untouched to avoid double injection.
- The command can process one file with `--html` or all `.html` files below a directory with `--dir`.
- Google credentials are read from env/file/ADC and are never written to audit logs.

## References

Google documents manual Google tag installation as placing the tag on every page immediately after the `<head>` element, and describes Google Tag Manager as the broader tag-management option for Google and third-party tags.
