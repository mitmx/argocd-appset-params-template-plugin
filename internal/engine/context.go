package engine

import "github.com/mitmx/argocd-values-pipeline-plugin/internal/maps"

type templateContextInput struct {
	Values      map[string]any
	Context     map[string]any
	Emit        map[string]any
	Passthrough map[string]any
	RawInput    map[string]any
	RequestMeta requestMeta
}

func buildTemplateContext(input templateContextInput) map[string]any {
	root := maps.CloneMap(input.Values)
	root["values"] = maps.CloneMap(input.Values)
	root["_values"] = maps.CloneMap(input.Values)
	root["context"] = maps.CloneMap(input.Context)
	root["emit"] = maps.CloneMap(input.Emit)
	root["_request"] = map[string]any{
		"applicationSetName": input.RequestMeta.ApplicationSetName,
		"inputValues":        maps.CloneMap(input.RequestMeta.InputValues),
	}
	root["_input"] = maps.CloneMap(input.RawInput)

	for _, layer := range []map[string]any{input.Passthrough, input.Emit, input.Context} {
		for k, v := range maps.CloneMap(layer) {
			if _, exists := root[k]; !exists {
				root[k] = v
			}
		}
	}

	if input.RequestMeta.ApplicationSetName != "" {
		if _, exists := root["applicationSetName"]; !exists {
			root["applicationSetName"] = input.RequestMeta.ApplicationSetName
		}
	}
	return root
}
