package main

import "time"

type CatalogType string

const (
	CatalogAgents CatalogType = "agents"
	CatalogSkills CatalogType = "skills"
	CatalogTools  CatalogType = "tools"
	CatalogModels CatalogType = "models"
)

type CatalogEntry struct {
	Name        string                 `json:"name"`
	Type        CatalogType            `json:"type"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
}

type RegisterRequest struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type CatalogStats struct {
	Agents int `json:"agents"`
	Skills int `json:"skills"`
	Tools  int `json:"tools"`
	Models int `json:"models"`
	Total  int `json:"total"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
