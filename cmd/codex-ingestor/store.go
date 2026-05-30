package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

type JobStore struct {
	mu   sync.RWMutex
	jobs map[string]IngestJob
	docs map[string]ProcessedDocument
}

func NewJobStore() *JobStore {
	return &JobStore{
		jobs: make(map[string]IngestJob),
		docs: make(map[string]ProcessedDocument),
	}
}

func (s *JobStore) CreateJob(docCount int) IngestJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := newID("job")
	job := IngestJob{
		ID:        id,
		Status:    "processing",
		StartedAt: time.Now().UTC(),
	}
	s.jobs[id] = job
	return job
}

func (s *JobStore) CompleteJob(id string, processed, chunks, embeddings int, errs []string) (IngestJob, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return IngestJob{}, false
	}
	now := time.Now().UTC()
	job.Status = "completed"
	if len(errs) > 0 {
		job.Status = "completed_with_errors"
	}
	job.DocumentsProcessed = processed
	job.ChunksGenerated = chunks
	job.EmbeddingsGenerated = embeddings
	job.Errors = errs
	job.CompletedAt = &now
	s.jobs[id] = job
	return job, true
}

func (s *JobStore) FailJob(id string, errMsg string) (IngestJob, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return IngestJob{}, false
	}
	now := time.Now().UTC()
	job.Status = "failed"
	job.Errors = append(job.Errors, errMsg)
	job.CompletedAt = &now
	s.jobs[id] = job
	return job, true
}

func (s *JobStore) GetJob(id string) (IngestJob, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	return job, ok
}

func (s *JobStore) ListJobs() []IngestJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]IngestJob, 0, len(s.jobs))
	for _, j := range s.jobs {
		out = append(out, j)
	}
	return out
}

func (s *JobStore) IngestDocument(req IngestRequest) (ProcessedDocument, error) {
	if strings.TrimSpace(req.Content) == "" {
		return ProcessedDocument{}, fmt.Errorf("content is required")
	}
	if req.Source == "" {
		return ProcessedDocument{}, fmt.Errorf("source is required")
	}

	chunks := chunkContent(req.Content)
	embedding := generateEmbedding(req.Content)

	doc := ProcessedDocument{
		ID:        newID("doc"),
		Content:   req.Content,
		Source:    req.Source,
		Metadata:  req.Metadata,
		Connector: req.Connector,
		Chunks:    chunks,
		Embedding: embedding,
	}

	s.mu.Lock()
	s.docs[doc.ID] = doc
	s.mu.Unlock()

	return doc, nil
}

func chunkContent(content string) []string {
	const chunkSize = 256
	words := strings.Fields(content)
	if len(words) == 0 {
		return []string{content}
	}

	chunks := make([]string, 0)
	var current strings.Builder
	for _, w := range words {
		if current.Len()+len(w)+1 > chunkSize && current.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteByte(' ')
		}
		current.WriteString(w)
	}
	if current.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}
	if len(chunks) == 0 {
		return []string{content}
	}
	return chunks
}

func generateEmbedding(content string) []float64 {
	vec := make([]float64, 16)
	lower := strings.ToLower(content)
	for i, c := range lower {
		idx := i % 16
		vec[idx] += float64(c%31+1) / 100.0
	}
	return vec
}

func newID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

func (s *JobStore) ProcessBatch(docs []IngestRequest) (IngestJob, []string) {
	job := s.CreateJob(len(docs))
	var docIDs []string
	var errs []string
	totalChunks := 0

	for i, req := range docs {
		doc, err := s.IngestDocument(req)
		if err != nil {
			errs = append(errs, fmt.Sprintf("document %d: %s", i, err.Error()))
			continue
		}
		docIDs = append(docIDs, doc.ID)
		totalChunks += len(doc.Chunks)
	}

	completed, _ := s.CompleteJob(job.ID, len(docIDs), totalChunks, len(docIDs), errs)
	return completed, docIDs
}

func (s *JobStore) ProcessSingle(req IngestRequest) (IngestJob, ProcessedDocument, error) {
	job := s.CreateJob(1)
	doc, err := s.IngestDocument(req)
	if err != nil {
		_, _ = s.FailJob(job.ID, err.Error())
		return IngestJob{}, ProcessedDocument{}, err
	}
	completed, _ := s.CompleteJob(job.ID, 1, len(doc.Chunks), 1, nil)
	return completed, doc, nil
}
