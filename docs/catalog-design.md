# Writing effective command catalogs

A useful catalog is more than a collection of shell aliases. It should make commands discoverable and guide users through the values they need.

## Use intent-focused titles

Prefer:

```yaml
title: Download GitHub Actions run logs
```

over:

```yaml
title: github-run-logs
```

The internal `name` can remain compact, while `title` should describe the action.

## Write descriptions that answer “when should I use this?”

```yaml
description: Download and extract logs from a GitHub Actions workflow run URL
```

Avoid repeating the title without adding context.

## Treat labels as semantic aliases

Use labels for alternative vocabulary:

```yaml
labels:
  - github
  - actions
  - workflow
  - logs
  - download
  - troubleshooting
```

For migrated aliases, preserve the old alias as a label:

```yaml
labels:
  - kubernetes
  - pods
  - logs
  - klo
```

## Group mechanical command variants

Do not create many nearly identical commands for:

```text
-o wide
-o yaml
-o json
--show-labels
--watch
```

Prefer one workflow with an options variable.

```yaml
- name: get-pods
  title: Get Kubernetes pods
  run: |
    display="{{display}}"

    if [[ "$display" == "standard" ]]; then
      display=""
    fi

    kubectl get pods -n "{{namespace}}" $display
```

## Separate commands by user intent

Keep these as distinct workflows:

```text
list pods
describe pod
follow logs
open shell
delete pod
```

They represent different user intentions even when they operate on the same resource.

## Prefer dynamic selectors

Instead of asking users to type values they can discover:

```yaml
- name: namespace
  source:
    type: command
    command: >-
      kubectl get namespaces
      -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'
```

Dynamic selectors reduce typing errors and improve discoverability.

## Keep dependencies linear

Declare broad scope before specific resources:

```text
context → namespace → pod → container
```

This matches how users narrow a target and keeps command sources easy to understand.

## Use multiline scripts for logic

Prefer a readable script:

```yaml
run: |
  set -euo pipefail

  output="{{output}}"
  mkdir -p "$output"

  # More steps...
```

over a dense one-liner.

## Add defensive shell behavior

For Bash scripts:

```bash
set -euo pipefail
```

Validate dangerous or structured values before acting on them.

## Avoid oversized catalogs

A curated catalog of guided workflows is usually more useful than hundreds of mechanical aliases.

A good command should:

- represent one clear user intention;
- have searchable metadata;
- expose changing values as variables;
- show a useful preview;
- avoid hidden destructive behavior.
