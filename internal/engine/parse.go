package engine

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mitmx/argocd-appset-params-template-plugin/internal/maps"
)

var allowedTopLevelKeys = map[string]struct{}{
	"context":           {},
	"params":            {},
	"defaults":          {},
	"overrides":         {},
	"selectorOverrides": {},
	"targets":           {},
	"options":           {},
}

var allowedTargetKeys = map[string]struct{}{
	"context":   {},
	"params":    {},
	"overrides": {},
}

var allowedOptionsKeys = map[string]struct{}{
	"maxTemplatePasses": {},
}

var allowedSelectorOverrideKeys = map[string]struct{}{
	"path":      {},
	"overrides": {},
	"default":   {},
}

func parseSpec(req Request, defaultMaxTemplatePasses int) (spec, error) {
	raw := maps.CloneMap(req.Parameters)
	if err := validateAllowedKeys(raw, allowedTopLevelKeys, "parameters"); err != nil {
		return spec{}, err
	}

	cfg := spec{
		Defaults:  map[string]any{},
		Overrides: map[string]any{},
		Context:   map[string]any{},
		Params:    map[string]any{},
		Options:   options{MaxTemplatePasses: defaultMaxTemplatePasses},
	}

	var err error
	if rawDefaults, exists := raw["defaults"]; exists {
		cfg.Defaults, err = requireMap(rawDefaults, "parameters.defaults")
		if err != nil {
			return spec{}, err
		}
	}
	if rawOverrides, exists := raw["overrides"]; exists {
		cfg.Overrides, err = requireMap(rawOverrides, "parameters.overrides")
		if err != nil {
			return spec{}, err
		}
	}
	if rawContext, exists := raw["context"]; exists {
		cfg.Context, err = requireMap(rawContext, "parameters.context")
		if err != nil {
			return spec{}, err
		}
	}
	if rawParams, exists := raw["params"]; exists {
		cfg.Params, err = requireMap(rawParams, "parameters.params")
		if err != nil {
			return spec{}, err
		}
	}
	if rawOptions, exists := raw["options"]; exists {
		optionsMap, err := requireMap(rawOptions, "parameters.options")
		if err != nil {
			return spec{}, err
		}
		if err := validateAllowedKeys(optionsMap, allowedOptionsKeys, "parameters.options"); err != nil {
			return spec{}, err
		}
		if v, ok := getInt(optionsMap, "maxTemplatePasses"); ok {
			cfg.Options.MaxTemplatePasses = v
		}
	}
	if cfg.Options.MaxTemplatePasses <= 0 {
		return spec{}, fmt.Errorf("parameters.options.maxTemplatePasses must be > 0")
	}

	cfg.SelectorOverrides, err = parseSelectorOverrides(raw)
	if err != nil {
		return spec{}, err
	}
	cfg.Targets, err = parseTargets(raw)
	if err != nil {
		return spec{}, err
	}
	if _, exists := raw["targets"]; exists && len(cfg.Targets) == 0 {
		return spec{}, fmt.Errorf("parameters.targets must not be empty")
	}

	return cfg, nil
}

func parseTargets(raw map[string]any) ([]target, error) {
	rawTargets, ok := raw["targets"]
	if !ok || rawTargets == nil {
		return nil, nil
	}
	list, ok := rawTargets.([]any)
	if !ok {
		return nil, fmt.Errorf("parameters.targets must be a list")
	}

	out := make([]target, 0, len(list))
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("parameters.targets[%d] must be an object", i)
		}
		if err := validateAllowedKeys(m, allowedTargetKeys, fmt.Sprintf("parameters.targets[%d]", i)); err != nil {
			return nil, err
		}

		t := target{Context: map[string]any{}, Params: map[string]any{}, Overrides: map[string]any{}}
		var err error
		if v, exists := m["context"]; exists {
			t.Context, err = requireMap(v, fmt.Sprintf("parameters.targets[%d].context", i))
			if err != nil {
				return nil, err
			}
		}
		if v, exists := m["params"]; exists {
			t.Params, err = requireMap(v, fmt.Sprintf("parameters.targets[%d].params", i))
			if err != nil {
				return nil, err
			}
		}
		if v, exists := m["overrides"]; exists {
			t.Overrides, err = requireMap(v, fmt.Sprintf("parameters.targets[%d].overrides", i))
			if err != nil {
				return nil, err
			}
		}
		out = append(out, t)
	}
	return out, nil
}

func parseSelectorOverrides(raw map[string]any) ([]selectorOverride, error) {
	rawSelectors, exists := raw["selectorOverrides"]
	if !exists || rawSelectors == nil {
		return nil, nil
	}

	list, ok := rawSelectors.([]any)
	if !ok {
		return nil, fmt.Errorf("parameters.selectorOverrides must be a list")
	}

	out := make([]selectorOverride, 0, len(list))
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("parameters.selectorOverrides[%d] must be an object", i)
		}
		if err := validateAllowedKeys(m, allowedSelectorOverrideKeys, fmt.Sprintf("parameters.selectorOverrides[%d]", i)); err != nil {
			return nil, err
		}

		path := strings.TrimSpace(getString(m, "path"))
		if path == "" {
			return nil, fmt.Errorf("parameters.selectorOverrides[%d].path is required", i)
		}
		if err := validateSelectorPath(path); err != nil {
			return nil, fmt.Errorf("parameters.selectorOverrides[%d].path: %w", i, err)
		}

		rawCases, exists := m["overrides"]
		if !exists {
			return nil, fmt.Errorf("parameters.selectorOverrides[%d].overrides is required", i)
		}
		casesMap, ok := rawCases.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("parameters.selectorOverrides[%d].overrides must be an object", i)
		}

		cases := make(map[string]map[string]any, len(casesMap))
		for key, value := range casesMap {
			caseMap, ok := value.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("parameters.selectorOverrides[%d].overrides[%q] must be an object", i, key)
			}
			cases[key] = maps.CloneMap(caseMap)
		}

		selector := selectorOverride{Path: path, Overrides: cases}
		if rawDefault, exists := m["default"]; exists {
			var err error
			selector.Default, err = requireMap(rawDefault, fmt.Sprintf("parameters.selectorOverrides[%d].default", i))
			if err != nil {
				return nil, err
			}
		}
		out = append(out, selector)
	}
	return out, nil
}

func validateSelectorPath(path string) error {
	for _, prefix := range []string{"context.", "params."} {
		if strings.HasPrefix(path, prefix) {
			segments := strings.Split(path, ".")
			for i, segment := range segments {
				if segment == "" {
					return fmt.Errorf("invalid empty path segment at position %d", i)
				}
			}
			if len(segments) < 2 {
				return fmt.Errorf("must reference a nested field")
			}
			return nil
		}
	}
	return fmt.Errorf("must start with \"context.\" or \"params.\"")
}

func validateAllowedKeys(m map[string]any, allowed map[string]struct{}, field string) error {
	for key := range m {
		if _, ok := allowed[key]; ok {
			continue
		}
		return fmt.Errorf("%s: unknown field %q", field, key)
	}
	return nil
}

func requireMap(v any, field string) (map[string]any, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an object", field)
	}
	return maps.CloneMap(m), nil
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
