# Retro: Google Analytics Provisioning

**PR:** #9 — Add Google analytics provisioning
**Merged:** 2026-05-26
**Branch:** feat/google-analytics-provisioning
**Design:** docs/plans/2026-05-26-google-analytics-provisioning-design.md
**Plan:** docs/plans/2026-05-26-google-analytics-provisioning.md
**Related ADRs:** none

## Adversarial-review findings, scored

| Phase | Finding | Severity | Outcome |
|---|---|---|---|
| design | Audit JSONL promised but not task-owned | Important | Resolved upfront |
| design | Live-apply blocker described but not verified | Important | Resolved upfront |
| design | GTM publish/versioning deferred | Minor | False positive |
| plan | Audit file not represented in task files | Important | Resolved upfront |
| plan | No-credentials live apply not verified | Important | Resolved upfront |
| plan | One PR is large | Minor | False positive |

## Gate misses

| Issue | Gate that missed | Why it slipped | Fix idea |
|---|---|---|---|
| Original plan used direct `go run ./cmd/... analytics ...`, but plugin CLI needs `--wfctl-cli` locally | writing-plans | Plan author copied the user-facing `wfctl` shape into a plugin-binary smoke command without checking SDK dispatch rules | Add plugin CLI smoke examples to writing guidance when `sdk.ServePluginFull` is present |
| CLI ensure path initially skipped default audit JSONL | requesting-code-review | No subagent tool was available, so inline review caught it late rather than as a separate review gate | Keep audit-path behavior in the review checklist for state-mutating plugin CLIs |

## Missed skill activations

| Gate | Fired? | Notes |
|---|---|---|
| brainstorming | yes | Design checkpoint presented; user approved autonomous continuation. |
| adversarial-design-review (design) | yes | Run inline; findings recorded in adversarial-review report. |
| writing-plans | yes | Plan created with scope manifest. |
| adversarial-design-review (plan) | yes | Run inline; findings recorded in adversarial-review report. |
| alignment-check | yes | Scope manifest check passed and plan was locked. |
| subagent-driven-development | partial | Codex subagent dispatch was unavailable; execution ran inline against the locked manifest. |
| finishing-a-development-branch | yes | PR #9 created and merged after green CI. |
| pr-monitoring | yes | PR and post-merge main CI were monitored until green. |
| post-merge-retrospective | yes | This retro. |

## What worked

- Scope lock caught the task-heading format issue before implementation.
- Adversarial review forced audit JSONL and live-apply blocking into the task plan before code.
- Runtime launch validation caught the local CLI dispatch shape before PR creation.
- PR monitoring confirmed PR CI and main-branch CI were green before closing the loop.

## What didn't

- The initial docs overemphasized CLI instead of YAML; user feedback corrected this before merge.
- The first implementation of ADC was ambiguous; inline review converted it to explicit `allow_adc`.
- The CLI audit path gap was caught late by inline review, not by an automated reviewer.

## Plugin-level follow-ups

- No plugin-level changes yet. If another plugin PR repeats the `--wfctl-cli` smoke mismatch, add a dedicated check to `runtime-launch-validation`.

## Project guidance updates

| Guidance file | Change | Reason |
|---|---|---|
| `docs/design-guidance.md` | no change | Existing guidance already says YAML pipeline is source of truth, secrets are declarative wiring, and audit trail is required for mutating ops. |
