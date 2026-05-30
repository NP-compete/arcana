package main

type HybridWeights struct {
	Semantic float64 `json:"semantic"`
	Keyword  float64 `json:"keyword"`
	Entity   float64 `json:"entity"`
}

type SemanticSearchRequest struct {
	Embedding []float64 `json:"embedding"`
	TopK      int       `json:"top_k"`
	ShardIDs  []string  `json:"shard_ids,omitempty"`
}

type KeywordSearchRequest struct {
	Query    string   `json:"query"`
	TopK     int      `json:"top_k"`
	ShardIDs []string `json:"shard_ids,omitempty"`
}

type EntitySearchRequest struct {
	Entities []string `json:"entities"`
	TopK     int      `json:"top_k"`
}

type HybridSearchRequest struct {
	Query     string        `json:"query"`
	Embedding []float64     `json:"embedding,omitempty"`
	Weights   HybridWeights `json:"weights"`
	TopK      int           `json:"top_k"`
	ShardIDs  []string      `json:"shard_ids,omitempty"`
}

type SearchHit struct {
	DocID    string            `json:"doc_id"`
	ShardID  string            `json:"shard_id"`
	Score    float64           `json:"score"`
	Snippet  string            `json:"snippet"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type SearchResponse struct {
	Hits      []SearchHit `json:"hits"`
	TotalHits int         `json:"total_hits"`
	SearchType string     `json:"search_type"`
}

type Document struct {
	ID        string
	ShardID   string
	Content   string
	Entities  []string
	Embedding []float64
	Authority float64
	Recency   float64
}
