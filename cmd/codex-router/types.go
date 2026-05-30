package main

type FusionWeights struct {
	Semantic  float64 `json:"semantic"`
	Temporal  float64 `json:"temporal"`
	Authority float64 `json:"authority"`
	Entity    float64 `json:"entity"`
}

type FusionProfile struct {
	Name    string        `json:"name"`
	Weights FusionWeights `json:"weights"`
}

type Shard struct {
	ID           string `json:"id"`
	Topic        string `json:"topic"`
	CentroidHash string `json:"centroid_hash"`
	DocCount     int    `json:"doc_count"`
}

type SearchRequest struct {
	Query   string            `json:"query"`
	Profile string            `json:"profile"`
	TopK    int               `json:"top_k"`
	Filters map[string]string `json:"filters,omitempty"`
}

type SearchHit struct {
	DocID    string            `json:"doc_id"`
	ShardID  string            `json:"shard_id"`
	Score    float64           `json:"score"`
	Snippet  string            `json:"snippet"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type SearchResult struct {
	Query       string      `json:"query"`
	Profile     string      `json:"profile"`
	Route       string      `json:"route"`
	ShardIDs    []string    `json:"shard_ids"`
	Results     []SearchHit `json:"results"`
	TotalHits   int         `json:"total_hits"`
	ElapsedMs   int64       `json:"elapsed_ms"`
}
