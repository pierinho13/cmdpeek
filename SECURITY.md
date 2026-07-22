# Security Policy

## Supported versions

Security fixes are applied to the latest released version of `cmdpeek`.

## Reporting a vulnerability

Please do not report security vulnerabilities through public GitHub issues.

Use GitHub's private vulnerability reporting feature for this repository when available.

Include:

- a clear description of the issue
- affected versions
- steps to reproduce
- potential impact
- any suggested mitigation

Please avoid including real Kubernetes Secret values, credentials, access tokens, kubeconfig files, private cluster information, or other sensitive data.
## Security considerations

`cmdpeek` executes local shell commands with the current user's environment and permissions.

## Configuration files are executable code

A configuration may:

- read or modify files;
- start processes;
- access the network;
- invoke credentials available to the user;
- run destructive commands.

Do not use untrusted configuration files.

## Variable interpolation

Variable values are currently inserted directly into command templates.

Example:

```yaml
run: git commit -m "{{message}}"
```

The surrounding quotes help with ordinary spaces, but they do not provide general shell-safe escaping for arbitrary input.

Review the final rendered command before confirming execution.

## Command source variables

A `command` variable runs its source command while resolving options.

```yaml
source:
  type: command
  command: kubectl get namespaces
```

This happens before the final command confirmation. Treat command sources as executable code as well.

## Sensitive values

Values may be visible in:

- the live command preview;
- the final confirmation screen;
- terminal scrollback;
- recordings and screenshots;
- process arguments;
- command output.

Avoid placing secrets directly in variables when possible. Prefer environment variables or credential-aware tools.

## Destructive commands

Use clear titles, descriptions and labels:

```yaml
title: Delete Kubernetes deployment
labels:
  - kubernetes
  - delete
  - destructive
```

Keep destructive operations separate from read-only workflows and make their effect obvious in the preview.
