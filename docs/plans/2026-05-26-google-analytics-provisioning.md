# Google Analytics Provisioning Implementation Plan

> **For the implementing agent:** REQUIRED SUB-SKILL: Use autodev:executing-plans to implement this plan task-by-task.

**Goal:** Add Google Analytics 4 and Google Tag Manager provisioning surfaces to the existing public Workflow analytics plugin.

**Architecture:** Keep `workflow-plugin-analytics` as the public plugin. Add a Google provider module, SDK-backed reconcile interfaces, dry-run/idempotent CLI and steps, audit JSONL, and gocodealone-multisite deployment guidance that stops before live apply.

**Tech Stack:** Go 1.26, `cloud.google.com/go/analytics/admin/apiv1alpha`, `google.golang.org/api/tagmanager/v2`, Workflow external plugin SDK.

**Base branch:** main

---

## Scope Manifest

**PR Count:** 1
**Tasks:** 6
**Estimated Lines of Change:** ~1100

**Out of scope:**
- GTM container version publishing.
- Automatic deletion/archive of Analytics or Tag Manager resources.
- Live Google API apply before operator credentials and account access exist.

**PR Grouping:**

| PR # | Title | Tasks | Branch |
|------|-------|-------|--------|
| 1 | Add Google analytics provisioning | Task 1, Task 2, Task 3, Task 4, Task 5, Task 6 | feat/google-analytics-provisioning |

**Status:** Draft

### Task 1: Google Provider Module and Credentials

**Files:**
- Modify: `internal/plugin.go`
- Create: `internal/google_provider.go`
- Create: `internal/google_provider_test.go`
- Create: `internal/google_audit.go`
- Create: `internal/google_audit_test.go`
- Modify: `plugin.json`

**Steps:**
1. Write failing tests that `analytics.google_provider` is exposed, registers config by module name, supports env/file/ADC credential inputs, redacts credential values in validation errors, and appends non-secret JSONL audit events.
2. Run `GOWORK=off go test ./internal -run 'TestGoogleProvider|TestPluginExposes' -count=1`; expected: FAIL because module type does not exist.
3. Implement `GoogleProviderConfig`, provider registry, module lifecycle, audit writer, and plugin `ModuleTypes/CreateModule`.
4. Run the focused tests; expected: PASS.
5. Rollback: revert this task commit to remove the module surface and plugin manifest entry.

### Task 2: GA4 Web Stream Reconciler

**Files:**
- Create: `internal/google_ga4.go`
- Create: `internal/google_ga4_test.go`

**Steps:**
1. Write failing tests for dry-run create plan, existing property reuse, existing stream reuse, create-property-then-create-stream, and invalid account/default URI validation.
2. Run `GOWORK=off go test ./internal -run TestEnsureGA4 -count=1`; expected: FAIL because reconciler does not exist.
3. Implement a fakeable `GA4AdminClient` interface plus SDK adapter using `cloud.google.com/go/analytics/admin/apiv1alpha`.
4. Implement `EnsureGA4WebStream` with list-before-create and operation summary output.
5. Run the focused tests; expected: PASS.
6. Rollback: revert this task commit; no live resources are touched by tests.

### Task 3: GTM Web Container Reconciler

**Files:**
- Create: `internal/google_gtm.go`
- Create: `internal/google_gtm_test.go`

**Steps:**
1. Write failing tests for dry-run create plan, existing container reuse, existing workspace reuse, optional GA4 config ensure, and invalid account/domain validation.
2. Run `GOWORK=off go test ./internal -run TestEnsureGTM -count=1`; expected: FAIL because reconciler does not exist.
3. Implement a fakeable `TagManagerClient` interface plus SDK adapter using `google.golang.org/api/tagmanager/v2`.
4. Implement `EnsureGTMWebContainer` with list-before-create and operation summary output.
5. Run the focused tests; expected: PASS.
6. Rollback: revert this task commit; no live resources are touched by tests.

### Task 4: CLI JSON Surfaces

**Files:**
- Modify: `internal/cli.go`
- Modify: `internal/cli_test.go`
- Modify: `README.md`
- Modify: `plugin.json`

**Steps:**
1. Write failing CLI tests for `analytics google ga4 ensure --dry-run` and `analytics google gtm ensure --dry-run` JSON output.
2. Run `GOWORK=off go test ./internal -run TestCLIAnalyticsGoogle -count=1`; expected: FAIL because subcommands do not exist.
3. Implement nested CLI parsing, JSON output, credential flags, and dry-run default examples.
4. Run the focused tests; expected: PASS.
5. Rollback: revert this task commit; existing `analytics inject` behavior remains in prior commits.

### Task 5: Workflow Ensure Steps

**Files:**
- Create: `internal/google_steps.go`
- Create: `internal/google_steps_test.go`
- Modify: `internal/plugin.go`
- Modify: `plugin.json`
- Modify: `plugin.contracts.json`

**Steps:**
1. Write failing tests that `step.analytics_google_ga4_ensure` and `step.analytics_google_gtm_ensure` execute dry-run through fake clients and output IDs/operations.
2. Run `GOWORK=off go test ./internal -run TestAnalyticsGoogle.*Step -count=1`; expected: FAIL because step types do not exist.
3. Implement step factories using the provider registry and per-step overrides.
4. Run the focused tests; expected: PASS.
5. Rollback: revert this task commit to remove provisioning step types.

### Task 6: Consumer Deployment Guidance and Verification

**Files:**
- Modify: `README.md`
- Create: `docs/gocodealone-multisite-google-analytics.md`
- Modify: `examples/minimal/config.yaml`

**Steps:**
1. Add docs for gocodealone-multisite plugin pin, Google secrets, GA/GTM dry-run commands, live apply blocker, and measurement ID injection flow.
2. Add an example Workflow config showing provider module, GA4 ensure step, GTM ensure step, and existing HTML injection step consuming returned IDs.
3. Run `GOWORK=off go test ./...`; expected: PASS.
4. Run `GOWORK=off go build ./...`; expected: PASS.
5. Run CLI smoke: `GOWORK=off go run ./cmd/workflow-plugin-analytics analytics google ga4 ensure --account accounts/123 --property-name example.com --stream-name example.com --default-uri https://example.com --dry-run`; expected JSON includes `"dry_run":true` and `"measurement_id":""`.
6. Confirm live apply is still blocked by running the same command without `--dry-run` and no credentials; expected non-zero exit and an error that names missing Google credentials without printing any credential value.
7. Rollback: revert docs/example commit; no live resources touched.
