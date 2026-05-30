package main

type FusionWeights struct {
	Semantic  float64 `json:"semantic"`
	Temporal  float64 `json:"temporal"`
	Authority float64 `json:"authority"`
	Entity    float64 `json:"entity"`
}

type ScoreRequest struct {
	Results []SearchHitInput `json:"results"`
	Profile string           `json:"profile"`
	Query   string           `json:"query"`
}

type SearchHitInput struct {
	DocID    string            `json:"doc_id"`
	ShardID  string            `json:"shard_id"`
	Score    float64           `json:"score"`
	Snippet  string            `json:"snippet"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type ScoredResult struct {
	DocID         string            `json:"doc_id"`
	ShardID       string            `json:"shard_id"`
	OriginalScore float64           `json:"original_score"`
	FusedScore    float64           `json:"fused_score"`
	Rank          int               `json:"rank"`
	Snippet       string            `json:"snippet"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Signals       SignalBreakdown   `json:"signals"`
}

type SignalBreakdown struct {
	Semantic  float64 `json:"semantic"`
	Temporal  float64 `json:"temporal"`
	Authority float64 `json:"authority"`
	Entity    float64 `json:"entity"`
}

type ScoreResponse struct {
	Query   string         `json:"query"`
	Profile string         `json:"profile"`
	Results []ScoredResult `json:"results"`
}

type ContradictionRequest struct {
	DocA string `json:"doc_a"`
	DocB string `json:"doc_b"`
}

type ContradictionReport struct {
	DocAID       string  `json:"doc_a_id"`
	DocBID       string  `json:"doc_b_id"`
	Contradicts  bool    `json:"contradicts"`
	Confidence   float64 `json:"confidence"`
	Explanation  string  `json:"explanation"`
}

type SupersessionRequest struct {
	DocA string `json:"doc_a"`
	DocB string `json:"doc_b"`
}

type SupersessionCheck struct {
	DocAID      string  `json:"doc_a_id"`
	DocBID      string  `json:"doc_b_id"`
	Supersedes  bool    `json:"supersedes"`
	Confidence  float64 `json:"confidence"`
	Explanation string  `json:"explanation"`
}

type DocumentRecord struct {
	Content   string
	Version   int
	UpdatedAt string
	Authority float64
}
