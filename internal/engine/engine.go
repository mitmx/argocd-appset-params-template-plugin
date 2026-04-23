package engine

import (
	"fmt"
	"strings"

	"github.com/mitmx/argocd-values-pipeline-plugin/internal/maps"
	"github.com/mitmx/argocd-values-pipeline-plugin/internal/render"
	"github.com/mitmx/argocd-values-pipeline-plugin/internal/serialize"
)

type Mode string

const (
	ModeResolve Mode = "resolve"
	ModeFanout  Mode = "fanout"
)

type Engine struct {
	defaultMaxTemplatePasses int
}

type Request struct {
	ApplicationSetName string
	Parameters         map[string]any
	Values             map[string]any
}

type spec struct {
	Mode              Mode
	Defaults          map[string]any
	Overrides         map[string]any
	SelectorOverrides []selectorOverride
	Context           map[string]any
	Emit              map[string]any
	Targets           []target
	Options           options
	Passthrough       map[string]any
	RawInput          map[string]any
	RequestMeta       requestMeta
}

type selectorOverride struct {
	Path      string
	Overrides map[string]map[string]any
}

type target struct {
	Defaults  map[string]any
	Overrides map[string]any
	Context   map[string]any
	Emit      map[string]any
}

type options struct {
	MaxTemplatePasses int
}

type requestMeta struct {
	ApplicationSetName string
	InputValues        map[string]any
}

func New(defaultMaxTemplatePasses int) Engine {
	if defaultMaxTemplatePasses <= 0 {
		defaultMaxTemplatePasses = 4
	}
	return Engine{defaultMaxTemplatePasses: defaultMaxTemplatePasses}
}

func (e Engine) Execute(req Request) ([]map[string]any, error) {
	cfg, err := parseSpec(req, e.defaultMaxTemplatePasses)
	if err != nil {
		return nil, err
	}

	items := cfg.Targets
	if cfg.Mode == ModeResolve && len(items) == 0 {
		items = []target{{}}
	}

	results := make([]map[string]any, 0, len(items))
	for idx, item := range items {
		resolved, err := e.resolveTarget(cfg, item)
		if err != nil {
			return nil, fmt.Errorf("target %d: %w", idx, err)
		}
		results = append(results, resolved)
	}
	return results, nil
}

func (e Engine) resolveTarget(cfg spec, item target) (map[string]any, error) {
	mergedContext := maps.MergeAll(cfg.Context, item.Context)
	mergedEmit := maps.MergeAll(cfg.Emit, item.Emit)

	baseValues := maps.MergeAll(cfg.Defaults, item.Defaults, cfg.Overrides)
	selectionContext := buildTemplateContext(templateContextInput{
		Values:      baseValues,
		Context:     mergedContext,
		Emit:        mergedEmit,
		Passthrough: cfg.Passthrough,
		RawInput:    cfg.RawInput,
		RequestMeta: cfg.RequestMeta,
	})

	selectedOverrides, err := applySelectorOverrides(cfg.SelectorOverrides, selectionContext)
	if err != nil {
		return nil, err
	}

	values := maps.MergeAll(baseValues, selectedOverrides, item.Overrides)
	renderer := render.New(cfg.Options.MaxTemplatePasses)
	resolvedValues, err := renderer.Resolve(values, func(current map[string]any) map[string]any {
		return buildTemplateContext(templateContextInput{
			Values:      current,
			Context:     mergedContext,
			Emit:        mergedEmit,
			Passthrough: cfg.Passthrough,
			RawInput:    cfg.RawInput,
			RequestMeta: cfg.RequestMeta,
		})
	})
	if err != nil {
		return nil, err
	}

	yamlText, err := serialize.MarshalYAML(resolvedValues)
	if err != nil {
		return nil, fmt.Errorf("marshal resolved values: %w", err)
	}

	out := maps.CloneMap(cfg.Passthrough)
	out = maps.DeepMerge(out, mergedEmit)
	out["context"] = maps.CloneMap(mergedContext)
	out["resolvedValues"] = resolvedValues
	out["resolvedValuesYaml"] = strings.TrimRight(yamlText, "\n")
	return out, nil
}

func applySelectorOverrides(selectors []selectorOverride, ctx map[string]any) (map[string]any, error) {
	result := map[string]any{}
	for _, selector := range selectors {
		value, ok, err := maps.LookupPath(ctx, selector.Path)
		if err != nil {
			return nil, fmt.Errorf("selector path %q: %w", selector.Path, err)
		}
		if !ok {
			continue
		}
		key, ok := maps.ToStringKey(value)
		if !ok {
			return nil, fmt.Errorf("selector path %q resolved to unsupported type %T", selector.Path, value)
		}
		override, ok := selector.Overrides[key]
		if !ok {
			continue
		}
		result = maps.DeepMerge(result, override)
	}
	return result, nil
}
