# Variable sources

`cmdpeek` supports four variable source types.

## Input

Use `input` for free-form values:

```yaml
- name: name
  prompt: Person name
  description: Name included in the greeting
  default: world
  source:
    type: input
```

The user may edit the default before continuing. Empty input is rejected.

## Options

Use `options` for a fixed list:

```yaml
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

Option values must not be empty.

For a logical “no additional flags” choice, use an explicit sentinel such as `standard` or `default`, then handle it in the script:

```yaml
run: |
  display="{{display}}"

  if [[ "$display" == "standard" ]]; then
    display=""
  fi

  kubectl get pods $display
```

## Environment

Use `environment` to initialize a value from an environment variable:

```yaml
- name: profile
  prompt: AWS profile
  default: default
  source:
    type: environment
    variable: AWS_PROFILE
```

If the environment variable is not set, `default` is used.

The resulting value remains editable.

## Command

Use `command` to generate selectable options:

```yaml
- name: branch
  prompt: Select Git branch
  source:
    type: command
    command: git branch --format='%(refname:short)'
```

Behavior:

- output is split by line;
- surrounding whitespace is removed;
- empty lines are discarded;
- duplicate values are removed;
- the command times out after 10 seconds;
- exactly one result is accepted automatically;
- multiple results open a searchable, paginated selector.

## Dependent variables

A command source may reference variables declared earlier:

```yaml
variables:
  - name: namespace
    source:
      type: command
      command: >-
        kubectl get namespaces
        -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'

  - name: pod
    source:
      type: command
      command: >-
        kubectl get pods -n "{{namespace}}"
        -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'
```

The declaration order matters. This is invalid:

```yaml
variables:
  - name: pod
    source:
      type: command
      command: kubectl get pods -n "{{namespace}}"

  - name: namespace
    source:
      type: input
```

## Selector controls

For `options` and multi-result `command` variables:

```text
/         Filter options
↑ / ↓     Navigate
← / →     Change page
Enter     Select
r         Retry command source
Esc       Clear filter or cancel
```

`r` is available for command-backed selectors.

## Preview behavior

The command preview updates with the current value:

```text
Ctrl+↑    Scroll preview up
Ctrl+↓    Scroll preview down
```

Unresolved variables are displayed as:

```text
<variable_name>
```
