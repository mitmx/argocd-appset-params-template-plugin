package engine

import (
	"fmt"

	"github.com/mitmx/argocd-appset-params-template-plugin/internal/maps"
	"github.com/mitmx/argocd-appset-params-template-plugin/internal/render"
)

type Engine struct {
	defaultMaxTemplatePasses int
}

type Request struct {
	Parameters map[string]any
}

type spec struct {
	Defaults          map[string]any
	Overrides         map[string]any
	SelectorOverrides []selectorOverride
	Context           map[string]any
	Params            map[string]any
	Targets           []target
	Options           options
}

type selectorOverride struct {
	Path      string
	Overrides map[string]map[string]any
	Default   map[string]any
}

type target struct {
	Context   map[string]any
	Params    map[string]any
	Overrides map[string]any
}

type options struct {
	MaxTemplatePasses int
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
		return nil, fmt.Errorf("parse: %w", err)
	}

	items := cfg.Targets
	if len(items) == 0 {
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
	mergedParams := maps.MergeAll(cfg.Params, item.Params)

	selectedOverrides, err := applySelectorOverrides(cfg.SelectorOverrides, mergedContext, mergedParams)
	if err != nil {
		return nil, fmt.Errorf("selector: %w", err)
	}

	result := maps.MergeAll(cfg.Defaults, cfg.Overrides, selectedOverrides, item.Overrides)
	renderer := render.New(cfg.Options.MaxTemplatePasses)
	resolvedResult, err := renderer.Resolve(result, func(current map[string]any) map[string]any {
		return buildTemplateContext(templateContextInput{
			Result:  current,
			Context: mergedContext,
			Params:  mergedParams,
			Target:  targetTemplateContext{Context: item.Context, Params: item.Params, Overrides: item.Overrides},
		})
	})
	if err != nil {
		return nil, fmt.Errorf("render: %w", err)
	}

	return map[string]any{
		"context": maps.CloneMap(mergedContext),
		"params":  maps.CloneMap(mergedParams),
		"result":  resolvedResult,
	}, nil
}

func applySelectorOverrides(selectors []selectorOverride, contextValues, paramsValues map[string]any) (map[string]any, error) {
	selectionRoot := map[string]any{
		"context": maps.CloneMap(contextValues),
		"params":  maps.CloneMap(paramsValues),
	}

	result := map[string]any{}
	for _, selector := range selectors {
		value, ok, err := maps.LookupPath(selectionRoot, selector.Path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", selector.Path, err)
		}
		if !ok {
			continue
		}
		key, ok := maps.ToStringKey(value)
		if !ok {
			return nil, fmt.Errorf("%s resolved to unsupported type %T", selector.Path, value)
		}
		if override, ok := selector.Overrides[key]; ok {
			result = maps.DeepMerge(result, override)
			continue
		}
		if len(selector.Default) > 0 {
			result = maps.DeepMerge(result, selector.Default)
		}
	}
	return result, nil
}
