# Configuration patterns

## Basic command

```yaml
- name: hello
  title: Say hello
  labels:
    - example
  run: echo "Hello from cmdpeek"
```

## Input with a default

```yaml
- name: greet
  title: Greet a person
  run: echo "Hello, {{name}}"
  variables:
    - name: name
      prompt: Person name
      default: world
      source:
        type: input
```

## Static environment selection

```yaml
- name: deploy
  title: Deploy application
  run: ./deploy.sh "{{environment}}"
  variables:
    - name: environment
      prompt: Select environment
      default: staging
      source:
        type: options
        values:
          - development
          - staging
          - production
```

## Value from the environment

```yaml
- name: show-aws-profile
  title: Show AWS profile
  run: echo "{{profile}}"
  variables:
    - name: profile
      prompt: AWS profile
      default: default
      source:
        type: environment
        variable: AWS_PROFILE
```

## Dynamic Git branch selector

```yaml
- name: checkout-branch
  title: Checkout Git branch
  run: git checkout "{{branch}}"
  variables:
    - name: branch
      prompt: Select branch
      source:
        type: command
        command: git branch --format='%(refname:short)'
```

## Dependent Kubernetes selectors

```yaml
- name: describe-pod
  title: Describe Kubernetes pod
  run: kubectl describe pod "{{pod}}" -n "{{namespace}}"
  variables:
    - name: namespace
      prompt: Select namespace
      source:
        type: command
        command: >-
          kubectl get namespaces
          -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'

    - name: pod
      prompt: Select pod
      source:
        type: command
        command: >-
          kubectl get pods -n "{{namespace}}"
          -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'
```

## Optional display behavior using a sentinel

Empty options are invalid. Use a named option instead:

```yaml
- name: display
  prompt: Select display mode
  default: standard
  source:
    type: options
    values:
      - standard
      - wide
      - yaml
```

```yaml
run: |
  display="{{display}}"

  case "$display" in
    standard) display="" ;;
    wide) display="-o wide" ;;
    yaml) display="-o yaml" ;;
  esac

  kubectl get pods $display
```

## Long script

```yaml
- name: generate-report
  title: Generate report
  run: |
    set -euo pipefail

    project="{{project}}"
    output="{{output}}"

    mkdir -p "$(dirname "$output")"

    {
      echo "Project: $project"
      echo "Generated at: $(date)"
    } > "$output"

    echo "Report written to: $output"
```

See also:

- [`../examples/basic.yaml`](../examples/basic.yaml)
- [`../examples/kubernetes.yaml`](../examples/kubernetes.yaml)
