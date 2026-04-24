package concepts

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"
)

func HydrateValues(concept *Concept, values map[string]any) (map[string]any, error) {
	if concept == nil {
		return nil, fmt.Errorf("concept is nil")
	}
	ret := map[string]any{}
	for k, v := range values {
		ret[k] = v
	}
	for _, param := range concept.Params {
		value, ok := ret[param.Name]
		if !ok || value == nil || value == "" {
			if param.Default != nil {
				value = param.Default
			} else if param.Required {
				return nil, fmt.Errorf("required param %q is missing", param.Name)
			} else {
				value = zeroValue(param.Type)
			}
		}
		coerced, err := coerceValue(param, value)
		if err != nil {
			return nil, err
		}
		ret[param.Name] = coerced
	}
	return ret, nil
}

func RenderConcept(concept *Concept, values map[string]any) (string, error) {
	if concept == nil {
		return "", fmt.Errorf("concept is nil")
	}
	hydrated, err := HydrateValues(concept, values)
	if err != nil {
		return "", err
	}
	funcs := template.FuncMap{
		"value":       func(name string) any { return hydrated[name] },
		"sqlString":   sqlString,
		"sqlLike":     sqlLike,
		"sqlStringIn": sqlStringIn,
		"sqlIntIn":    sqlIntIn,
	}
	tmpl, err := template.New(concept.Name).Option("missingkey=zero").Funcs(funcs).Parse(concept.Query)
	if err != nil {
		return "", fmt.Errorf("parse concept SQL template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, hydrated); err != nil {
		return "", fmt.Errorf("render concept SQL: %w", err)
	}
	return strings.TrimSpace(buf.String()), nil
}

func zeroValue(t ParamType) any {
	switch t {
	case ParamInt:
		return 0
	case ParamBool:
		return false
	case ParamStringList:
		return []string{}
	case ParamIntList:
		return []int{}
	default:
		return ""
	}
}

func coerceValue(param Param, value any) (any, error) {
	switch param.Type {
	case ParamString, ParamChoice:
		v := fmt.Sprint(value)
		if param.Type == ParamChoice && v != "" {
			ok := false
			for _, choice := range param.Choices {
				if v == choice {
					ok = true
					break
				}
			}
			if !ok {
				return nil, fmt.Errorf("param %q value %q is not one of %s", param.Name, v, strings.Join(param.Choices, ", "))
			}
		}
		return v, nil
	case ParamInt:
		return toInt(param.Name, value)
	case ParamBool:
		return toBool(param.Name, value)
	case ParamStringList:
		return toStringList(value), nil
	case ParamIntList:
		return toIntList(param.Name, value)
	default:
		return nil, fmt.Errorf("param %q has unsupported type %q", param.Name, param.Type)
	}
}

func toInt(name string, value any) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		if strings.TrimSpace(v) == "" {
			return 0, nil
		}
		i, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, fmt.Errorf("param %q expects int: %w", name, err)
		}
		return i, nil
	default:
		return 0, fmt.Errorf("param %q expects int, got %T", name, value)
	}
}

func toBool(name string, value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		if strings.TrimSpace(v) == "" {
			return false, nil
		}
		b, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return false, fmt.Errorf("param %q expects bool: %w", name, err)
		}
		return b, nil
	default:
		return false, fmt.Errorf("param %q expects bool, got %T", name, value)
	}
}

func toStringList(value any) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		ret := make([]string, 0, len(v))
		for _, item := range v {
			ret = append(ret, fmt.Sprint(item))
		}
		return ret
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		parts := strings.Split(v, ",")
		ret := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				ret = append(ret, trimmed)
			}
		}
		return ret
	default:
		return []string{fmt.Sprint(value)}
	}
}

func toIntList(name string, value any) ([]int, error) {
	strings := toStringList(value)
	ret := make([]int, 0, len(strings))
	for _, s := range strings {
		i, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("param %q expects int list: %w", name, err)
		}
		ret = append(ret, i)
	}
	return ret, nil
}

func sqlString(value any) string {
	return "'" + strings.ReplaceAll(fmt.Sprint(value), "'", "''") + "'"
}

func sqlLike(value any) string {
	return sqlString("%" + fmt.Sprint(value) + "%")
}

func sqlStringIn(value any) string {
	items := toStringList(value)
	if len(items) == 0 {
		return "''"
	}
	quoted := make([]string, 0, len(items))
	for _, item := range items {
		quoted = append(quoted, sqlString(item))
	}
	return strings.Join(quoted, ", ")
}

func sqlIntIn(value any) (string, error) {
	items, err := toIntList("value", value)
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "0", nil
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, strconv.Itoa(item))
	}
	return strings.Join(parts, ", "), nil
}
