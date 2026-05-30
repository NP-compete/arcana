package main

import (
	"math"
	"sort"
	"strings"
	"sync"
)

var fusionProfiles = map[string]FusionWeights{
	"default":         {Semantic: 0.35, Temporal: 0.25, Authority: 0.25, Entity: 0.15},
	"semantic_first":  {Semantic: 0.65, Temporal: 0.10, Authority: 0.15, Entity: 0.10},
	"navigation_link": {Semantic: 0.15, Temporal: 0.10, Authority: 0.35, Entity: 0.40},
	"recent_first":    {Semantic: 0.20, Temporal: 0.55, Authority: 0.15, Entity: 0.10},
}

type ScorerStore struct {
	mu   sync.RWMutex
	docs map[string]DocumentRecord
}

func NewScorerStore() *ScorerStore {
	return &ScorerStore{docs: seedDocRecords()}
}

func seedDocRecords() map[string]DocumentRecord {
	return map[string]DocumentRecord{
		"doc-platform-001": {
			Content:   "Arcana supports Kubernetes-native agent deployment with CRD-based configuration.",
			Version:   3, UpdatedAt: "2026-05-01", Authority: 0.95,
		},
		"doc-platform-002": {
			Content:   "Arcana requires Docker-only deployment without Kubernetes support.",
			Version:   1, UpdatedAt: "2024-01-15", Authority: 0.40,
		},
		"doc-codex-001": {
			Content:   "Codex search uses hybrid fusion with configurable profiles.",
			Version:   2, UpdatedAt: "2026-04-20", Authority: 0.90,
		},
		"doc-codex-002": {
			Content:   "Codex search supports only keyword BM25 with no vector search.",
			Version:   1, UpdatedAt: "2025-06-10", Authority: 0.55,
		},
		"doc-govern-001": {
			Content:   "Ward guardrails enforce policy via OPA before agent responses.",
			Version:   2, UpdatedAt: "2026-03-15", Authority: 0.88,
		},
		"doc-ops-001": {
			Content:   "Production deployment uses Helm charts on Kind clusters.",
			Version:   4, UpdatedAt: "2026-05-28", Authority: 0.82,
		},
	}
}

func (s *ScorerStore) EnsureDoc(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.docs[id]; !ok {
		s.docs[id] = DocumentRecord{
			Content:   "Generic document content for " + id,
			Version:   1,
			UpdatedAt: "2026-01-01",
			Authority: 0.5,
		}
	}
}

func (s *ScorerStore) GetDoc(id string) (DocumentRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.docs[id]
	return d, ok
}

func (s *ScorerStore) ScoreResults(query, profile string, hits []SearchHitInput) []ScoredResult {
	weights, ok := fusionProfiles[profile]
	if !ok {
		weights = fusionProfiles["default"]
		profile = "default"
	}
	if profile == "" {
		profile = "default"
		weights = fusionProfiles["default"]
	}

	scored := make([]ScoredResult, 0, len(hits))
	for _, h := range hits {
		s.EnsureDoc(h.DocID)
		doc, _ := s.GetDoc(h.DocID)

		sem := semanticSignal(query, h.Snippet)
		temp := temporalSignal(doc.UpdatedAt)
		auth := doc.Authority
		ent := entitySignal(query, h.Metadata)

		fused := weights.Semantic*sem + weights.Temporal*temp + weights.Authority*auth + weights.Entity*ent
		fused = (fused + h.Score) / 2.0

		scored = append(scored, ScoredResult{
			DocID:         h.DocID,
			ShardID:       h.ShardID,
			OriginalScore: round4(h.Score),
			FusedScore:    round4(fused),
			Snippet:       h.Snippet,
			Metadata:      h.Metadata,
			Signals: SignalBreakdown{
				Semantic:  round4(sem),
				Temporal:  round4(temp),
				Authority: round4(auth),
				Entity:    round4(ent),
			},
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].FusedScore > scored[j].FusedScore
	})
	for i := range scored {
		scored[i].Rank = i + 1
	}
	return scored
}

func semanticSignal(query, snippet string) float64 {
	qTerms := strings.Fields(strings.ToLower(query))
	if len(qTerms) == 0 {
		return 0
	}
	snippetLower := strings.ToLower(snippet)
	matches := 0
	for _, t := range qTerms {
		if strings.Contains(snippetLower, t) {
			matches++
		}
	}
	return float64(matches) / float64(len(qTerms))
}

func temporalSignal(updatedAt string) float64 {
	switch updatedAt {
	case "2026-05-28", "2026-05-01":
		return 0.95
	case "2026-04-20", "2026-03-15":
		return 0.80
	case "2025-06-10":
		return 0.55
	case "2024-01-15":
		return 0.30
	default:
		return 0.60
	}
}

func entitySignal(query string, metadata map[string]string) float64 {
	if metadata == nil {
		return 0.2
	}
	q := strings.ToLower(query)
	hits := 0
	for _, v := range metadata {
		if strings.Contains(q, strings.ToLower(v)) {
			hits++
		}
	}
	if hits == 0 {
		return 0.15
	}
	return math.Min(1.0, float64(hits)*0.35)
}

func (s *ScorerStore) CheckContradiction(docA, docB string) ContradictionReport {
	s.EnsureDoc(docA)
	s.EnsureDoc(docB)
	a, _ := s.GetDoc(docA)
	b, _ := s.GetDoc(docB)

	contradicts, confidence, explanation := analyzeContradiction(a.Content, b.Content)
	return ContradictionReport{
		DocAID:      docA,
		DocBID:      docB,
		Contradicts: contradicts,
		Confidence:  round4(confidence),
		Explanation: explanation,
	}
}

func analyzeContradiction(a, b string) (bool, float64, string) {
	aLower := strings.ToLower(a)
	bLower := strings.ToLower(b)

	pairs := []struct {
		pos, neg string
		topic    string
	}{
		{"kubernetes", "docker-only", "deployment platform"},
		{"hybrid fusion", "only keyword", "search capabilities"},
		{"supports", "without", "feature support"},
		{"vector search", "no vector", "vector search"},
	}

	for _, p := range pairs {
		aHasPos := strings.Contains(aLower, p.pos)
		aHasNeg := strings.Contains(aLower, p.neg)
		bHasPos := strings.Contains(bLower, p.pos)
		bHasNeg := strings.Contains(bLower, p.neg)

		if (aHasPos && bHasNeg) || (aHasNeg && bHasPos) {
			return true, 0.85, "Documents contain opposing statements about " + p.topic
		}
	}

	overlap := tokenOverlap(aLower, bLower)
	if overlap < 0.15 {
		return false, 0.70, "Documents cover different topics with no detected conflict"
	}
	return false, 0.55, "Documents discuss related topics without explicit contradiction"
}

func (s *ScorerStore) CheckSupersession(docA, docB string) SupersessionCheck {
	s.EnsureDoc(docA)
	s.EnsureDoc(docB)
	a, _ := s.GetDoc(docA)
	b, _ := s.GetDoc(docB)

	supersedes, confidence, explanation := analyzeSupersession(a, b)
	return SupersessionCheck{
		DocAID:      docA,
		DocBID:      docB,
		Supersedes:  supersedes,
		Confidence:  round4(confidence),
		Explanation: explanation,
	}
}

func analyzeSupersession(older, newer DocumentRecord) (bool, float64, string) {
	if newer.Version <= older.Version {
		return false, 0.60, "doc_b does not have a higher version than doc_a"
	}
	if newer.UpdatedAt <= older.UpdatedAt {
		return false, 0.55, "doc_b is not more recent than doc_a"
	}

	overlap := tokenOverlap(strings.ToLower(older.Content), strings.ToLower(newer.Content))
	if overlap < 0.20 {
		return false, 0.65, "Documents cover different topics; supersession unlikely"
	}

	if newer.Authority >= older.Authority {
		conf := 0.70 + float64(newer.Version-older.Version)*0.05
		if conf > 0.95 {
			conf = 0.95
		}
		return true, conf, "doc_b is a newer, higher-authority revision covering the same topic"
	}
	return false, 0.50, "doc_b is newer but lacks sufficient authority to supersede doc_a"
}

func tokenOverlap(a, b string) float64 {
	aTokens := tokenSet(a)
	bTokens := tokenSet(b)
	if len(aTokens) == 0 || len(bTokens) == 0 {
		return 0
	}
	intersect := 0
	for t := range aTokens {
		if bTokens[t] {
			intersect++
		}
	}
	union := len(aTokens) + len(bTokens) - intersect
	if union == 0 {
		return 0
	}
	return float64(intersect) / float64(union)
}

func tokenSet(text string) map[string]bool {
	tokens := make(map[string]bool)
	for _, w := range strings.Fields(text) {
		if len(w) > 3 {
			tokens[w] = true
		}
	}
	return tokens
}

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}
