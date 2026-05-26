# gocodealone-multisite Google Analytics Provisioning

This runbook wires `gocodealone-multisite` to programmatically provision GA4 web streams and optional GTM web containers through `workflow-plugin-analytics`.

The source of truth should be `deploy.yaml` / `deploy.prereq.yaml`, not ad hoc CLI state. CLI commands below are smoke probes for the same operations expressed in YAML.

## Prerequisites

- Enable Google Analytics Admin API and Tag Manager API for the credential project.
- Create or choose a service account / ADC principal.
- Grant it access to the target Google Analytics account(s) and Google Tag Manager account(s).
- Store credentials as a deploy secret, for example `GOOGLE_ANALYTICS_ADMIN_CREDENTIALS_JSON`.
- Choose umbrella accounts:
  - GA: `accounts/<analytics-account-id>`
  - GTM: `accounts/<tag-manager-account-id>`

Stop here until the operator has provisioned API access.

## wfctl plugin pin

Add the analytics plugin to `gocodealone-multisite/wfctl.yaml` after release:

```yaml
plugins:
  - name: workflow-plugin-analytics
    version: vNEXT
    source: github.com/GoCodeAlone/workflow-plugin-analytics
```

## Secret entries

Add this entry to `deploy.prereq.yaml` and `deploy.yaml` secret lists:

```yaml
- name: GOOGLE_ANALYTICS_ADMIN_CREDENTIALS_JSON
  description: Service-account JSON with Analytics Admin and Tag Manager access.
```

## deploy.yaml desired state

Add the provider module and `pipelines.apply` steps to `deploy.yaml` so analytics resources are reconciled with the rest of the deploy intent:

```yaml
modules:
  - name: google-analytics
    type: analytics.google_provider
    config:
      credentials_json: ${GOOGLE_ANALYTICS_ADMIN_CREDENTIALS_JSON}
      # Keep false for GitHub-secret driven deploys; set true only when
      # the runner's ADC identity is intentionally the deploy principal.
      allow_adc: false
      analytics_account: accounts/123456789
      tag_manager_account: accounts/987654321

pipelines:
  apply:
    steps:
      - name: ensure_gocodealone_ga4
        type: step.analytics_google_ga4_ensure
        config:
          provider: google-analytics
          # Existing GA4 property created manually for GoCodeAlone.
          property: properties/538139248
          property_name: GoCodeAlone
          stream_name: gocodealone.tech
          default_uri: https://gocodealone.tech
          dry_run: true

      - name: ensure_gocodealone_gtm
        type: step.analytics_google_gtm_ensure
        config:
          provider: google-analytics
          container_name: gocodealone.tech
          domains: [gocodealone.tech, www.gocodealone.tech]
          workspace_name: workflow
          measurement_id: ${ensure_gocodealone_ga4.measurement_id}
          dry_run: true
```

Keep `dry_run: true` until the operator grants API access. After reviewing dry-run output, switch the specific site step to `dry_run: false` and run:

```sh
wfctl infra apply -c deploy.yaml --wait
```

## CLI smoke probe

Run the equivalent GA4 dry-run directly when validating credentials or debugging:

```sh
wfctl analytics google ga4 ensure \
  --account accounts/395146029 \
  --property properties/538139248 \
  --property-name GoCodeAlone \
  --stream-name gocodealone.tech \
  --default-uri https://gocodealone.tech \
  --credentials-json-env GOOGLE_ANALYTICS_ADMIN_CREDENTIALS_JSON \
  --dry-run
```

Optional GTM dry-run:

```sh
wfctl analytics google gtm ensure \
  --account accounts/987654321 \
  --container-name gocodealone.tech \
  --domain gocodealone.tech \
  --domain www.gocodealone.tech \
  --measurement-id G-XXXXXXXXXX \
  --credentials-json-env GOOGLE_ANALYTICS_ADMIN_CREDENTIALS_JSON \
  --dry-run
```

The output is JSON and can be copied into deployment logs or a future Workflow pipeline step. Live apply removes `--dry-run`, but only after the dry-run output is reviewed.

## Content repo update

After live GA4 apply returns a measurement ID, set it in the content repo `multisite.yaml`:

```yaml
analytics:
  google:
    measurement_id: G-XXXXXXXXXX
    anonymize_ip: true
```

`gocodealone-multisite` then passes that ID into `step.analytics_inject_html`; tenants without an ID keep analytics disabled.

## Rollback

- Remove or blank `analytics.google.measurement_id` in the content repo and redeploy content.
- Run `wfctl analytics inject` with an empty tag ID when mutating static assets to remove managed snippets.
- Do not delete GA/GTM resources automatically; deletion can destroy useful history.
