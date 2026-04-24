package concepts

import (
	"fmt"
	"io/fs"
	"strings"
)

const PreambleMarker = "codebase-browser concept"

type ParamType string

const (
	ParamString     ParamType = "string"
	ParamInt        ParamType = "int"
	ParamBool       ParamType = "bool"
	ParamChoice     ParamType = "choice"
	ParamStringList ParamType = "stringList"
	ParamIntList    ParamType = "intList"
)

type Param struct {
	Name      string    `yaml:"name"`
	Type      ParamType `yaml:"type"`
	Help      string    `yaml:"help,omitempty"`
	Required  bool      `yaml:"required,omitempty"`
	Default   any       `yaml:"default,omitempty"`
	Choices   []string  `yaml:"choices,omitempty"`
	ShortFlag string    `yaml:"shortFlag,omitempty"`
}

type ConceptSpec struct {
	Name   string   `yaml:"name"`
	Short  string   `yaml:"short"`
	Long   string   `yaml:"long,omitempty"`
	Tags   []string `yaml:"tags,omitempty"`
	Params []Param  `yaml:"params,omitempty"`
	Query  string   `yaml:"-"`
}

type Concept struct {
	Name       string
	Folder     string
	Path       string
	Short      string
	Long       string
	Tags       []string
	Params     []Param
	Query      string
	SourceRoot string
	SourcePath string
}

type SourceRoot struct {
	Name    string
	FS      fs.FS
	RootDir string
}

type Catalog struct {
	Concepts []*Concept
	ByPath   map[string]*Concept
	ByName   map[string]*Concept
}

func (s *ConceptSpec) Validate() error {
	if s == nil {
		return fmt.Errorf("concept spec is nil")
	}
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("concept name is required")
	}
	if strings.TrimSpace(s.Short) == "" {
		return fmt.Errorf("concept short description is required")
	}
	if strings.TrimSpace(s.Query) == "" {
		return fmt.Errorf("concept query is required")
	}
	seen := map[string]struct{}{}
	for _, param := range s.Params {
		if err := param.Validate(); err != nil {
			return fmt.Errorf("param %q: %w", param.Name, err)
		}
		key := strings.TrimSpace(param.Name)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate param %q", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func (p Param) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.ContainsAny(p.Name, " \t\n\r") {
		return fmt.Errorf("name must not contain whitespace")
	}
	switch p.Type {
	case "":
		return fmt.Errorf("type is required")
	case ParamString, ParamInt, ParamBool, ParamChoice, ParamStringList, ParamIntList:
	default:
		return fmt.Errorf("unsupported type %q", p.Type)
	}
	if p.Type == ParamChoice && len(p.Choices) == 0 {
		return fmt.Errorf("choice params require choices")
	}
	return nil
}
