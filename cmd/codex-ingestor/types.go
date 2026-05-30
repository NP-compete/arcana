package main

import "time"

type IngestRequest struct {
	Content   string            `json:"content"`
	Source    string            `json:"source"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Connector string            `json:"connector,omitempty"`
}

type BatchIngestRequest struct {
	Documents []IngestRequest `json:"documents"`
}

type IngestJob struct {
	ID                 string    `json:"id"`
	Status             string    `json:"status"`
	DocumentsProcessed int       `json:"documents_processed"`
	Errors             []string  `json:"errors,omitempty"`
	StartedAt          time.Time `json:"started_at"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	ChunksGenerated    int       `json:"chunks_generated"`
	EmbeddingsGenerated int      `json:"embeddings_generated"`
}

type IngestResponse struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ProcessedDocument struct {
	ID        string
	Content   string
	Source    string
	Metadata  map[string]string
	Connector string
	Chunks    []string
	Embedding []float64
}
