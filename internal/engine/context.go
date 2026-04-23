package engine

import "github.com/mitmx/argocd-appset-params-template-plugin/internal/maps"

type templateContextInput struct {
	Result  map[string]any
	Context map[string]any
	Params  map[string]any
	Target  targetTemplateContext
}

type targetTemplateContext struct {
	Context   map[string]any
	Params    map[string]any
	Overrides map[string]any
}

func buildTemplateContext(input templateContextInput) map[string]any {
	return map[string]any{
		"result":  maps.CloneMap(input.Result),
		"context": maps.CloneMap(input.Context),
		"params":  maps.CloneMap(input.Params),
		"target": map[string]any{
			"context":   maps.CloneMap(input.Target.Context),
			"params":    maps.CloneMap(input.Target.Params),
			"overrides": maps.CloneMap(input.Target.Overrides),
		},
	}
}
