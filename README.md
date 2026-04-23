# argocd-appset-params-template-plugin

ApplicationSet plugin generator for one job: take structured input, merge it, render templates inside it, and return parameter objects for `ApplicationSet.template`.

It does **not** generate `ApplicationSet` manifests.
It does **not** replace Argo CD multiple sources.
It stays compatible with both:

- one Helm source with `toYaml .result`
- multiple Helm sources with `toYaml (.result.<part> | default dict)`

## Why this exists

Plain ApplicationSet templating is good at wiring generators into a template.
It is not a recursive structured data pipeline.

This plugin fills that gap:

- global `defaults`
- global `overrides`
- selector-based overrides by `context.*` or `params.*`
- optional fanout through `targets`
- recursive template rendering inside the final result
- pure-expression type preservation

The main use case is Ansible-like templated defaults for Helm values, without hardcoding chart-specific logic into the plugin.

## Contract

All input lives under:

```yaml
spec.generators[].plugin.input.parameters
```

Supported top-level keys:

```yaml
context: {}
params: {}
defaults: {}
overrides: {}
selectorOverrides:
  - path: context.cluster.name
    overrides:
      dev-blabla:
        domain: dev.example.com
    default:
      domain: default.example.com
targets:
  - context: {}
    params: {}
    overrides: {}
options:
  maxTemplatePasses: 4
```

Strict rules:

- unknown top-level keys are rejected
- unknown target keys are rejected
- `selectorOverrides[*].path` must start with `context.` or `params.`
- `targets[].defaults` is not supported
- if `targets` is omitted, the plugin returns one result
- if `targets` is present, the plugin returns one result per target

## Output

Each generated object looks like this:

```yaml
context: {}
params: {}
result: {}
```

Use it in `ApplicationSet.template` however you need.

Single Helm source:

```yaml
helm:
  values: |
    {{- toYaml .result | nindent 12 }}
```

Multiple Helm sources inside one `Application`:

```yaml
sources:
  - repoURL: ...
    helm:
      values: |
        {{- toYaml (.result.operator | default dict) | nindent 12 }}

  - repoURL: ...
    helm:
      values: |
        {{- toYaml (.result.cluster | default dict) | nindent 12 }}
```

## Merge order

For one resolved item:

1. root `defaults`
2. root `overrides`
3. matching `selectorOverrides` in declaration order
4. target `overrides`
5. recursive template rendering until stable

`context` and `params` are merged separately:

1. root `context` + target `context`
2. root `params` + target `params`

Merge semantics:

- map + map -> recursive merge
- list + list -> replace
- scalar + scalar -> replace
- `null` stays `null`

## Template context

Template strings inside the final result get exactly these namespaces:

```gotemplate
{{ .result.domain }}
{{ .context.cluster.name }}
{{ .params.appName }}
{{ .target.params.appName }}
```

No root aliases like `.name`, `.domain`, or `.appName`.

## Type preservation

There are two rendering modes for strings:

- **pure expression**: the whole string is one template action like `{{ 3 }}` or `{{ dict "tier" "frontend" }}`
- **mixed string**: anything else like `svc-{{ .result.port }}`

Rules:

- pure expressions preserve type
- mixed strings always stay strings

Examples:

```gotemplate
{{ 3 }}                              -> int
{{ true }}                           -> bool
{{ list 80 443 }}                    -> []any{80, 443}
{{ dict "tier" "frontend" }}       -> map[string]any
{{ .result.someString }}             -> string
svc-{{ .result.port }}               -> string
```

To force a string in a pure expression, use `quote`.

## Template functions

Built-ins plus:

- `default`
- `lower`, `upper`, `trim`, `trimPrefix`, `trimSuffix`
- `replace`, `contains`, `hasPrefix`, `hasSuffix`
- `quote`, `squote`
- `list`, `dict`
- `join`, `split`
- `toJson`, `toPrettyJson`, `toYaml`
- `indent`, `nindent`
- `normalize`
- `required`

## Patterns

### 1. One app per cluster via label

Use `matrix(clusters, plugin)`.

See `manifests/applicationset-label-per-cluster-example.yaml`.

### 2. Multiple instances on one cluster

Use `matrix(list(instances), plugin)` when the instance list is explicit.

See `manifests/applicationset-multi-instance-example.yaml`.

### 3. One `Application`, multiple Helm charts

Let the plugin return one structured `result`, then split it in the template per source.

See `manifests/applicationset-multi-source-compatible-example.yaml`.

## Deploy

Runtime manifests are in `manifests/`.

They assume:

- the controller and the plugin share the same token stored in `argocd-appset-params-template-plugin-token`
- the plugin is reachable at `http://argocd-appset-params-template-plugin.argocd.svc.cluster.local`

Apply:

```bash
kubectl apply -k manifests
```
