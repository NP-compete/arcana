package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
)

type scoredDoc struct {
	doc   Document
	score float64
}

type DocumentStore struct {
	mu   sync.RWMutex
	docs map[string]Document
}

func NewDocumentStore() *DocumentStore {
	docs := seedDocuments()
	return &DocumentStore{docs: docs}
}

func seedDocuments() map[string]Document {
	raw := []struct {
		id, shard, content string
		entities           []string
		authority, recency float64
	}{
		{
			id: "doc-platform-001", shard: "shard-platform",
			content: "Arcana is a Kubernetes-native AI agent platform with LangGraph orchestration and MCP tool integration.",
			entities: []string{"Arcana", "Kubernetes", "LangGraph", "MCP"}, authority: 0.95, recency: 0.88,
		},
		{
			id: "doc-platform-002", shard: "shard-platform",
			content: "The agent mesh gateway supports A2A and ACP protocols for multi-agent communication.",
			entities: []string{"A2A", "ACP", "mesh"}, authority: 0.82, recency: 0.75,
		},
		{
			id: "doc-codex-001", shard: "shard-codex",
			content: "Codex provides semantic vector search with BM25 keyword fusion and entity matching.",
			entities: []string{"Codex", "BM25", "embedding"}, authority: 0.90, recency: 0.92,
		},
		{
			id: "doc-codex-002", shard: "shard-codex",
			content: "Fusion profiles control weight distribution across semantic, temporal, authority, and entity signals.",
			entities: []string{"fusion", "semantic_first", "recent_first"}, authority: 0.85, recency: 0.80,
		},
		{
			id: "doc-govern-001", shard: "shard-govern",
			content: "Arcana Ward applies guardrails using OPA policies and content filtering pipelines.",
			entities: []string{"Ward", "OPA", "guardrails"}, authority: 0.88, recency: 0.70,
		},
		{
			id: "doc-ops-001", shard: "shard-ops",
			content: "Deploy Arcana on Kind with Helm charts, Temporal workflows, and PostgreSQL backing stores.",
			entities: []string{"Kind", "Helm", "Temporal", "PostgreSQL"}, authority: 0.78, recency: 0.95,
		},
		{
			id: "doc-ops-002", shard: "shard-ops",
			content: "Redis and NATS provide caching and event streaming for the ops plane.",
			entities: []string{"Redis", "NATS"}, authority: 0.72, recency: 0.85,
		},
		{
			id: "doc-connectors-001", shard: "shard-codex",
			content: "Tier 1 connectors include GitHub, GitLab, Confluence, Slack, Notion, and Snowflake.",
			entities: []string{"GitHub", "GitLab", "Confluence", "Slack"}, authority: 0.80, recency: 0.78,
		},
	}

	docs := make(map[string]Document, len(raw))
	for _, r := range raw {
		docs[r.id] = Document{
			ID:        r.id,
			ShardID:   r.shard,
			Content:   r.content,
			Entities:  r.entities,
			Embedding: textToEmbedding(r.content),
			Authority: r.authority,
			Recency:   r.recency,
		}
	}
	return docs
}

func textToEmbedding(text string) []float64 {
	words := strings.Fields(strings.ToLower(text))
	vec := make([]float64, 16)
	for i, w := range words {
		for j, c := range w {
			idx := (int(c) + j + i) % 16
			vec[idx] += 1.0 / float64(len(w)+1)
		}
	}
	norm := 0.0
	for _, v := range vec {
		norm += v * v
	}
	if norm > 0 {
		norm = math.Sqrt(norm)
		for i := range vec {
			vec[i] /= norm
		}
	}
	return vec
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var dot, na, nb float64
	for i := 0; i < n; i++ {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func bm25Score(query, content string) float64 {
	qTerms := strings.Fields(strings.ToLower(query))
	contentLower := strings.ToLower(content)
	score := 0.0
	k1, b := 1.2, 0.75
	avgDL := 20.0
	dl := float64(len(strings.Fields(content)))
	for _, term := range qTerms {
		tf := 0.0
		for _, w := range strings.Fields(contentLower) {
			if w == term {
				tf++
			}
		}
		if tf == 0 {
			continue
		}
		idf := math.Log(1 + (8.0-tf+0.5)/(tf+0.5))
		tfNorm := (tf * (k1 + 1)) / (tf + k1*(1-b+b*dl/avgDL))
		score += idf * tfNorm
	}
	return score
}

func (s *DocumentStore) filterByShards(shardIDs []string) []Document {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(shardIDs) == 0 {
		out := make([]Document, 0, len(s.docs))
		for _, d := range s.docs {
			out = append(out, d)
		}
		return out
	}

	allowed := make(map[string]struct{}, len(shardIDs))
	for _, id := range shardIDs {
		allowed[id] = struct{}{}
	}
	out := make([]Document, 0)
	for _, d := range s.docs {
		if _, ok := allowed[d.ShardID]; ok {
			out = append(out, d)
		}
	}
	return out
}

func (s *DocumentStore) SemanticSearch(embedding []float64, topK int, shardIDs []string) []SearchHit {
	if topK <= 0 {
		topK = 10
	}
	if len(embedding) == 0 {
		embedding = textToEmbedding("default query")
	}

	candidates := s.filterByShards(shardIDs)
	scoredDocs := make([]scoredDoc, 0, len(candidates))
	for _, d := range candidates {
		scoredDocs = append(scoredDocs, scoredDoc{d, cosineSimilarity(embedding, d.Embedding)})
	}
	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].score > scoredDocs[j].score
	})

	return toHits(scoredDocs, topK, "semantic")
}

func (s *DocumentStore) KeywordSearch(query string, topK int, shardIDs []string) []SearchHit {
	if topK <= 0 {
		topK = 10
	}
	candidates := s.filterByShards(shardIDs)
	scoredDocs := make([]scoredDoc, 0, len(candidates))
	for _, d := range candidates {
		scoredDocs = append(scoredDocs, scoredDoc{d, bm25Score(query, d.Content)})
	}
	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].score > scoredDocs[j].score
	})
	return toHits(scoredDocs, topK, "keyword")
}

func (s *DocumentStore) EntitySearch(entities []string, topK int) []SearchHit {
	if topK <= 0 {
		topK = 10
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	entitySet := make(map[string]struct{}, len(entities))
	for _, e := range entities {
		entitySet[strings.ToLower(e)] = struct{}{}
	}

	scoredDocs := make([]scoredDoc, 0)
	for _, d := range s.docs {
		matches := 0
		for _, ent := range d.Entities {
			if _, ok := entitySet[strings.ToLower(ent)]; ok {
				matches++
			}
		}
		if matches > 0 {
			scoredDocs = append(scoredDocs, scoredDoc{d, float64(matches) / float64(len(d.Entities))})
		}
	}
	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].score > scoredDocs[j].score
	})
	return toHits(scoredDocs, topK, "entity")
}

func (s *DocumentStore) HybridSearch(query string, embedding []float64, weights HybridWeights, topK int, shardIDs []string) []SearchHit {
	if topK <= 0 {
		topK = 10
	}
	if weights.Semantic == 0 && weights.Keyword == 0 && weights.Entity == 0 {
		weights = HybridWeights{Semantic: 0.5, Keyword: 0.35, Entity: 0.15}
	}
	if len(embedding) == 0 {
		embedding = textToEmbedding(query)
	}

	candidates := s.filterByShards(shardIDs)
	scoredDocs := make([]scoredDoc, 0, len(candidates))
	for _, d := range candidates {
		sem := cosineSimilarity(embedding, d.Embedding)
		kw := bm25Score(query, d.Content)
		entScore := 0.0
		for _, ent := range d.Entities {
			if strings.Contains(strings.ToLower(query), strings.ToLower(ent)) {
				entScore += 1.0 / float64(len(d.Entities))
			}
		}
		combined := weights.Semantic*sem + weights.Keyword*normalizeBM25(kw) + weights.Entity*entScore
		scoredDocs = append(scoredDocs, scoredDoc{d, combined})
	}
	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].score > scoredDocs[j].score
	})
	return toHits(scoredDocs, topK, "hybrid")
}

func normalizeBM25(score float64) float64 {
	if score <= 0 {
		return 0
	}
	return score / (score + 1.0)
}

func toHits(scoredDocs []scoredDoc, topK int, searchType string) []SearchHit {
	hits := make([]SearchHit, 0, topK)
	for i, sd := range scoredDocs {
		if i >= topK || sd.score <= 0 {
			break
		}
		snippet := sd.doc.Content
		if len(snippet) > 120 {
			snippet = snippet[:117] + "..."
		}
		hits = append(hits, SearchHit{
			DocID:   sd.doc.ID,
			ShardID: sd.doc.ShardID,
			Score:   roundScore(sd.score),
			Snippet: snippet,
			Metadata: map[string]string{
				"search_type": searchType,
				"authority":   formatFloat(sd.doc.Authority),
				"recency":     formatFloat(sd.doc.Recency),
			},
		})
	}
	return hits
}

func roundScore(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func formatFloat(v float64) string {
	return strings.TrimRight(strings.TrimRight(
		strings.Replace(fmt.Sprintf("%.4f", v), ".0000", "", 1), "0"), ".")
}
