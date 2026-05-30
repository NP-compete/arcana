package main

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type yamlBlueprint struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Nodes       []Node    `yaml:"nodes"`
	Edges       []Edge    `yaml:"edges"`
	Fallback    string    `yaml:"fallback"`
}

func ParseBlueprintYAML(content string) (*Blueprint, error) {
	var yb yamlBlueprint
	if err := yaml.Unmarshal([]byte(content), &yb); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}
	if strings.TrimSpace(yb.Name) == "" {
		return nil, fmt.Errorf("blueprint name is required")
	}
	if len(yb.Nodes) == 0 {
		return nil, fmt.Errorf("blueprint must have at least one node")
	}
	for i, n := range yb.Nodes {
		if n.ID == "" {
			return nil, fmt.Errorf("node at index %d missing id", i)
		}
		if n.Type == "" {
			return nil, fmt.Errorf("node %s missing type", n.ID)
		}
	}

	return &Blueprint{
		Name:        yb.Name,
		Description: yb.Description,
		Nodes:       yb.Nodes,
		Edges:       yb.Edges,
		Fallback:    yb.Fallback,
	}, nil
}
