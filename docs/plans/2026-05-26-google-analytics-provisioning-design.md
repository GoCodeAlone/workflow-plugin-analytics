# Google Analytics Provisioning Design

## Goal

Extend the existing public `workflow-plugin-analytics` plugin from HTML snippet injection into idempotent provisioning for Google Analytics 4 web streams and Google Tag Manager web containers.

The deploy target is Workflow/wfctl: site repos and apps declare the desired analytics resources, `wfctl analytics google ...` or Workflow steps reconcile them, and existing injection uses the returned GA measurement ID or GTM public container ID. Live apply is blocked until an operator provides Google API credentials and access to the target Analytics and Tag Manager accounts.

## Current State

- `workflow-plugin-analytics` is public (`plugin.json private: false`) and already supports GA4 `gtag.js` and GTM HTML injection.
- It has no Google SDK clients, no provider module, no provisioning CLI, no state/audit file, and no way to create/find GA properties, GA web data streams, GTM containers, workspaces, or Google tag configs.
- `gocodealone-multisite` already models `multisite.yaml.analytics.google.measurement_id`; that field currently assumes the ID was manually created elsewhere.

## Global Design Guidance

Source: `/Users/jon/workspace/docs/design-guidance.md`

| guidance | design response |
|---|---|
| Workflow platform is the substrate; no standalone tools | Extend `workflow-plugin-analytics` with plugin modules, steps, and `wfctl` CLI passthrough. |
| Reuse over rebuild | Extend the existing analytics plugin instead of creating separate GA/GTM repos. |
| Primary language Go; official SDKs isolated | Use official Google Go SDK packages behind small internal interfaces. |
| Secrets never logged | Credentials are loaded from env/file/module config and audit output records only resource IDs. |
| Audit trail for state-mutating ops | Append JSONL audit events under `${XDG_STATE_HOME:-$HOME/.local/state}/wfctl/plugins/analytics/google-audit.jsonl`. |
| Cost discipline; live cloud tests opt-in | Unit tests use fake SDK interfaces. Live reconciliation is gated by explicit credentials and command invocation. |
| End-to-end via real consumer | Add gocodealone-multisite deploy/runbook steps showing the plugin pin and dry-run/apply sequence, stopping before live apply. |

## References

- Google Analytics Admin Go client exposes `AnalyticsAdminClient.CreateProperty`, `ListProperties`, `CreateDataStream`, and `ListDataStreams` in `cloud.google.com/go/analytics/admin/apiv1alpha`.
- Google Tag Manager Go client exposes `tagmanager.NewService`, container, workspace, and `GtagConfig` services in `google.golang.org/api/tagmanager/v2`.
- GA web streams expose output `measurement_id`; GTM containers expose output `publicId`.

## Approaches

1. **Manual IDs plus existing injection only.**
   - Pros: no new code.
   - Cons: does not satisfy programmatic provisioning or tracking; repeats manual console work per site.
2. **New separate provider plugins (`workflow-plugin-google-analytics`, `workflow-plugin-google-tag-manager`).**
   - Pros: narrow repositories.
   - Cons: fragments an already public analytics plugin and makes shared inject/provision flows harder.
3. **Extend `workflow-plugin-analytics` with Google provisioning.**
   - Pros: one public analytics surface, preserves current injection, lets provisioning and injection share provider naming and validation.
   - Cons: plugin grows beyond HTML mutation into cloud API reconciliation.

Chosen: option 3. The responsibility is still "analytics/tag-manager", and the existing plugin is already the public integration point.

## Design

### Provider Module

Add module type `analytics.google_provider`.

Config:

```yaml
modules:
  - name: google-analytics
    type: analytics.google_provider
    config:
      credentials_json_env: GOOGLE_ANALYTICS_ADMIN_CREDENTIALS_JSON
      credentials_file_env: GOOGLE_APPLICATION_CREDENTIALS
      analytics_account: accounts/123456789
      tag_manager_account: accounts/987654321
      audit_path: ""
```

The module registers a provider by name. Empty credentials use Google Application Default Credentials. Inline JSON and file paths are never logged. `analytics_account` and `tag_manager_account` can be overridden per step/CLI call to support umbrella accounts.

### GA4 Reconcile

Add `EnsureGA4WebStream(ctx, request)`:

- Validate account as `accounts/<id>`.
- List properties under `parent:<account>` and find exact `display_name`.
- Create the property only if missing.
- List web data streams under the property and find exact `display_name` or matching default URI.
- Create the web data stream only if missing.
- Return resource names and measurement ID.

`dry_run` returns the intended operations without creating resources.

### GTM Reconcile

Add `EnsureGTMWebContainer(ctx, request)`:

- Validate GTM account as `accounts/<id>`.
- List containers and find exact `name` or matching domain set.
- Create a web container only if missing.
- Ensure a workspace exists by name.
- If `measurement_id` is present, ensure a Google tag config exists in that workspace for GA4.
- Return container ID, public ID, workspace path, and operation list.

Publishing versions is out of scope for the first pass. The operator can inspect/publish in GTM, or a later task can add version submit/publish once the first live reconcile proves the account model.

### CLI

Extend the existing `analytics` command:

```sh
wfctl analytics google ga4 ensure \
  --account accounts/123456789 \
  --property-name gocodealone.tech \
  --stream-name gocodealone.tech \
  --default-uri https://gocodealone.tech \
  --credentials-json-env GOOGLE_ANALYTICS_ADMIN_CREDENTIALS_JSON \
  --dry-run

wfctl analytics google gtm ensure \
  --account accounts/987654321 \
  --container-name gocodealone.tech \
  --domain gocodealone.tech \
  --measurement-id G-XXXXXXXXXX \
  --credentials-json-env GOOGLE_ANALYTICS_ADMIN_CREDENTIALS_JSON \
  --dry-run
```

The CLI prints JSON output so deploy jobs can capture returned IDs.

### Workflow Steps

Add:

- `step.analytics_google_ga4_ensure`
- `step.analytics_google_gtm_ensure`

Both steps accept module name, account override, names/domains, `dry_run`, and credentials overrides. They return structured IDs and operation summaries for downstream injection.

### gocodealone-multisite Deployment Shape

Add runbook/config guidance, not live apply:

1. Add `workflow-plugin-analytics` to `wfctl.yaml`.
2. Add Google API secret entries in `deploy.prereq.yaml`/`deploy.yaml`.
3. Add a pre-deploy dry-run command that ensures a GA4 property/web stream for each known site domain.
4. Add optional GTM dry-run if the site wants container-managed tags.
5. After operator provisions API access, run the dry-run and inspect JSON output.
6. Only then run live apply and update each content repo `multisite.yaml.analytics.google.measurement_id`.

The first implementation documents these commands and stops before running live Google API calls.

## Security Review

- Credentials are read from env/file/ADC and passed to Google SDK options only.
- Errors must not echo credential values.
- Audit JSONL records timestamp, action, dry-run flag, account/resource names, and operation names, not secrets.
- Least privilege requires Google Analytics Admin access to target GA accounts and GTM account access for container management.
- The plugin does not expose public HTTP endpoints; abuse risk is deploy/operator-side credential misuse.

## Infrastructure Impact

Creates Google Analytics properties/web data streams and Google Tag Manager containers/workspaces/configs when live apply is used. These are third-party SaaS resources, not DigitalOcean infrastructure. Live mutation requires explicit operator credentials and should run from controlled deploy environments.

## Multi-Component Validation

- Unit tests use fake GA/GTM clients to prove idempotent list-before-create behavior.
- CLI tests run dry-run commands and assert JSON output.
- Step tests load the provider module and execute ensure steps against fake clients.
- Consumer validation is a gocodealone-multisite deploy/runbook dry-run. Live apply remains blocked until API access exists.

## Assumptions

- The Google service account or ADC principal can be granted access to the Analytics and GTM accounts.
- Exact display names are acceptable idempotency keys for first pass.
- A GA4 web stream measurement ID is sufficient for existing injection.
- GTM publish/versioning can wait until after initial container/workspace/config provisioning is proven.

## Self-Challenge

1. Laziest solution: keep a spreadsheet of IDs and inject env vars. Rejected because the user asked to provision and track IDs programmatically across many sites.
2. Fragile assumption: display-name matching may collide. Mitigation: require account scoping, return existing IDs, and document that names should be domain-like and unique.
3. YAGNI risk: GTM config creation might be too much for pass one. Kept because the ask explicitly includes GTM and programmatic management; publish/versioning is deferred.

## Rollback

- Revert the plugin pin/config in consumer apps.
- Re-run injection with empty tag/container IDs to remove managed snippets.
- Do not auto-delete Google resources in rollback; deletion risks historical data loss. Operators can archive/delete manually after audit review.
- Revert this plugin PR if provisioning code breaks existing injection; existing tests cover injection compatibility.

## Deployment Blocker

Stop before live Google API mutation. Required operator inputs:

- Google Cloud project/API enablement for Analytics Admin API and Tag Manager API.
- Credential secret value or ADC setup.
- GA account IDs and GTM account IDs for each umbrella.
- Confirmation that the principal has create/list permissions in those accounts.

## Backport: YAML Primary Surface

2026-05-26: User clarified GA/GTM should be managed from `wfctl`/IaC/infra YAML together with secret management. The design already had Workflow steps, but examples overemphasized CLI. Corrected behavior: `deploy.yaml` or `infra.yaml` owns `analytics.google_provider` plus `pipelines.apply` ensure steps; CLI remains a smoke/operator path. Manifest scope unchanged because Task 5 and Task 6 already cover steps and consumer deployment guidance.
