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
	Agents            int `json:"agents"`
	Skills            int `json:"skills"`
	Tools             int `json:"tools"`
	Models            int `json:"models"`
	Total             int `json:"total"`
	PendingApprovals  int `json:"pending_approvals"`
	PublishedVersions int `json:"published_versions"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// PackageVersion tracks a single semantic version of a catalog package.
type PackageVersion struct {
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	Author    string    `json:"author"`
	Digest    string    `json:"digest"`
	Status    string    `json:"status"` // pending, approved, published, rejected
	Notes     string    `json:"notes,omitempty"`
}

// ApprovalRequest represents a pending review for a package version change.
type ApprovalRequest struct {
	ID          string    `json:"id"`
	PackageType string    `json:"package_type"`
	PackageName string    `json:"package_name"`
	Version     string    `json:"version"`
	Author      string    `json:"author"`
	Status      string    `json:"status"` // pending, approved, rejected
	SubmittedAt time.Time `json:"submitted_at"`
	ReviewedAt  time.Time `json:"reviewed_at,omitempty"`
	ReviewedBy  string    `json:"reviewed_by,omitempty"`
	Comment     string    `json:"comment,omitempty"`
	Diff        string    `json:"diff,omitempty"`
}
