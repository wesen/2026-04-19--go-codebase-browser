package concepts

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func LooksLikeConceptSQL(contents []byte) bool {
	s := strings.TrimLeft(string(contents), "\ufeff\r\n\t ")
	if !strings.HasPrefix(s, "/*") {
		return false
	}
	end := strings.Index(s, "*/")
	if end == -1 {
		return false
	}
	raw := strings.TrimSpace(s[2:end])
	return strings.HasPrefix(raw, PreambleMarker)
}

func ParseSQLConcept(path string, contents []byte) (*ConceptSpec, error) {
	metadata, body, err := splitConceptPreamble(contents)
	if err != nil {
		return nil, fmt.Errorf("parse concept preamble %s: %w", path, err)
	}
	var spec ConceptSpec
	if err := yaml.Unmarshal([]byte(metadata), &spec); err != nil {
		return nil, fmt.Errorf("decode concept metadata %s: %w", path, err)
	}
	spec.Query = strings.TrimSpace(body)
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("validate concept %s: %w", path, err)
	}
	return &spec, nil
}

func splitConceptPreamble(contents []byte) (string, string, error) {
	s := strings.TrimLeft(string(contents), "\ufeff\r\n\t ")
	if !strings.HasPrefix(s, "/*") {
		return "", "", fmt.Errorf("missing concept preamble")
	}
	end := strings.Index(s, "*/")
	if end == -1 {
		return "", "", fmt.Errorf("unterminated concept preamble")
	}
	raw := strings.TrimSpace(s[2:end])
	if !strings.HasPrefix(raw, PreambleMarker) {
		return "", "", fmt.Errorf("invalid concept preamble marker")
	}
	metadata := strings.TrimSpace(strings.TrimPrefix(raw, PreambleMarker))
	if metadata == "" {
		return "", "", fmt.Errorf("empty concept metadata")
	}
	body := strings.TrimSpace(s[end+2:])
	if body == "" {
		return "", "", fmt.Errorf("missing concept query")
	}
	return metadata, body, nil
}
