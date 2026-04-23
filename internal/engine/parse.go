package engine

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mitmx/argocd-values-pipeline-plugin/internal/maps"
)

var reservedTopLevelKeys = map[string]struct{}{
	"mode":               {},
	"context":            {},
	"emit":               {},
	"defaults":           {},
	"overrides":          {},
	"selectorOverrides":  {},
	"targets":            {},
	"options":            {},
	"maxPasses":          {},
	"overridesByCluster": {},
}

func parseSpec(req Request, defaultMaxTemplatePasses int) (spec, error) {
	raw := maps.CloneMap(req.Parameters)
	cfg := spec{
		Defaults:    map[string]any{},
		Overrides:   map[string]any{},
		Context:     map[string]any{},
		Emit:        map[string]any{},
		Passthrough: passthrough(raw),
		RawInput:    raw,
		RequestMeta: requestMeta{
			ApplicationSetName: req.ApplicationSetName,
			InputValues:        maps.CloneMap(req.Values),
		},
		Options: options{MaxTemplatePasses: defaultMaxTemplatePasses},
	}

	switch mode := strings.TrimSpace(getString(raw, "mode")); mode {
	case "", string(ModeResolve):
		cfg.Mode = ModeResolve
	case string(ModeFanout):
		cfg.Mode = ModeFanout
	default:
		return spec{}, fmt.Errorf("unsupported mode %q", mode)
	}

	var err error
	if rawDefaults, exists := raw["defaults"]; exists {
		cfg.Defaults, err = requireMap(rawDefaults, "defaults")
		if err != nil {
			return spec{}, err
		}
	}
	if rawOverrides, exists := raw["overrides"]; exists {
		cfg.Overrides, err = requireMap(rawOverrides, "overrides")
		if err != nil {
			return spec{}, err
		}
	}
	if rawContext, exists := raw["context"]; exists {
		cfg.Context, err = requireMap(rawContext, "context")
		if err != nil {
			return spec{}, err
		}
	}
	if rawEmit, exists := raw["emit"]; exists {
		cfg.Emit, err = requireMap(rawEmit, "emit")
		if err != nil {
			return spec{}, err
		}
	}
	if rawOptions, exists := raw["options"]; exists {
		optionsMap, err := requireMap(rawOptions, "options")
		if err != nil {
			return spec{}, err
		}
		if v, ok := getInt(optionsMap, "maxTemplatePasses"); ok {
			cfg.Options.MaxTemplatePasses = v
		}
	}
	if v, ok := getInt(raw, "maxPasses"); ok && cfg.Options.MaxTemplatePasses == defaultMaxTemplatePasses {
		cfg.Options.MaxTemplatePasses = v
	}
	if cfg.Options.MaxTemplatePasses <= 0 {
		return spec{}, fmt.Errorf("options.maxTemplatePasses must be > 0")
	}

	if legacyCluster := getMap(raw, "cluster"); len(legacyCluster) > 0 {
		if _, exists := cfg.Context["cluster"]; !exists {
			cfg.Context["cluster"] = legacyCluster
		}
	}

	cfg.SelectorOverrides, err = parseSelectorOverrides(raw)
	if err != nil {
		return spec{}, err
	}
	cfg.Targets, err = parseTargets(raw)
	if err != nil {
		return spec{}, err
	}

	if cfg.Mode == ModeResolve && len(cfg.Targets) > 1 {
		return spec{}, fmt.Errorf("mode resolve accepts at most one target")
	}
	if cfg.Mode == ModeFanout && len(cfg.Targets) == 0 {
		return spec{}, fmt.Errorf("mode fanout requires at least one target")
	}
	return cfg, nil
}

func passthrough(raw map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range raw {
		if _, reserved := reservedTopLevelKeys[k]; reserved {
			continue
		}
		out[k] = maps.DeepCopy(v)
	}
	return out
}

func parseTargets(raw map[string]any) ([]target, error) {
	rawTargets, ok := raw["targets"]
	if !ok || rawTargets == nil {
		return nil, nil
	}
	list, ok := rawTargets.([]any)
	if !ok {
		return nil, fmt.Errorf("targets must be a list")
	}
	out := make([]target, 0, len(list))
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("targets[%d] must be an object", i)
		}
		t := target{
			Defaults:  map[string]any{},
			Overrides: map[string]any{},
			Context:   map[string]any{},
			Emit:      map[string]any{},
		}
		var err error
		if v, exists := m["defaults"]; exists {
			t.Defaults, err = requireMap(v, fmt.Sprintf("targets[%d].defaults", i))
			if err != nil {
				return nil, err
			}
		}
		if v, exists := m["overrides"]; exists {
			t.Overrides, err = requireMap(v, fmt.Sprintf("targets[%d].overrides", i))
			if err != nil {
				return nil, err
			}
		}
		if v, exists := m["context"]; exists {
			t.Context, err = requireMap(v, fmt.Sprintf("targets[%d].context", i))
			if err != nil {
				return nil, err
			}
		}
		if v, exists := m["emit"]; exists {
			t.Emit, err = requireMap(v, fmt.Sprintf("targets[%d].emit", i))
			if err != nil {
				return nil, err
			}
		}
		out = append(out, t)
	}
	return out, nil
}

func parseSelectorOverrides(raw map[string]any) ([]selectorOverride, error) {
	out := []selectorOverride{}
	if rawSelectors, exists := raw["selectorOverrides"]; exists && rawSelectors != nil {
		list, ok := rawSelectors.([]any)
		if !ok {
			return nil, fmt.Errorf("selectorOverrides must be a list")
		}
		for i, item := range list {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("selectorOverrides[%d] must be an object", i)
			}
			path := strings.TrimSpace(getString(m, "path"))
			if path == "" {
				return nil, fmt.Errorf("selectorOverrides[%d].path is required", i)
			}
			rawCases, exists := m["overrides"]
			if !exists {
				return nil, fmt.Errorf("selectorOverrides[%d].overrides is required", i)
			}
			casesMap, ok := rawCases.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("selectorOverrides[%d].overrides must be an object", i)
			}
			cases := map[string]map[string]any{}
			for key, value := range casesMap {
				caseMap, ok := value.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("selectorOverrides[%d].overrides[%q] must be an object", i, key)
				}
				cases[key] = maps.CloneMap(caseMap)
			}
			out = append(out, selectorOverride{Path: path, Overrides: cases})
		}
	}
	if legacy := getMap(raw, "overridesByCluster"); len(legacy) > 0 {
		cases := map[string]map[string]any{}
		for key, value := range legacy {
			caseMap, ok := value.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("overridesByCluster[%q] must be an object", key)
			}
			cases[key] = maps.CloneMap(caseMap)
		}
		out = append(out,
			selectorOverride{Path: "name", Overrides: cases},
			selectorOverride{Path: "nameNormalized", Overrides: cases},
		)
	}
	return out, nil
}

func requireMap(v any, field string) (map[string]any, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an object", field)
	}
	return maps.CloneMap(m), nil
}

func getMap(m map[string]any, key string) map[string]any {
	if raw, ok := m[key]; ok {
		if typed, ok := raw.(map[string]any); ok {
			return maps.CloneMap(typed)
		}
	}
	return nil
}

func getString(m map[string]any, key string) string {
	if raw, ok := m[key]; ok {
		if typed, ok := raw.(string); ok {
			return typed
		}
	}
	return ""
}

func getInt(m map[string]any, key string) (int, bool) {
	if raw, ok := m[key]; ok {
		switch t := raw.(type) {
		case int:
			return t, true
		case int64:
			return int(t), true
		case float64:
			return int(t), true
		case string:
			v, err := strconv.Atoi(strings.TrimSpace(t))
			if err == nil {
				return v, true
			}
		}
	}
	return 0, false
}
