# Workflow Analytics Plugin

> ✅ **Verified** — used in production at **buymywishlist**. This plugin has been validated end-to-end in a merged main-branch wfctl.yaml of an active GoCodeAlone project.

`workflow-plugin-analytics` injects analytics and tag-manager snippets into rendered HTML assets from `wfctl`.

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

## Providers

- `google-analytics`: injects the Google tag into `<head>`.
- `google-tag-manager`: injects the GTM script into `<head>` and the noscript iframe immediately after `<body>`.

## Safety

- Only IDs containing letters, numbers, `_`, and `-` are accepted.
- Managed blocks are replaced idempotently.
- Existing unmanaged snippets for the same provider ID are detected and left untouched to avoid double injection.
- The command can process one file with `--html` or all `.html` files below a directory with `--dir`.

## References

Google documents manual Google tag installation as placing the tag on every page immediately after the `<head>` element, and describes Google Tag Manager as the broader tag-management option for Google and third-party tags.
