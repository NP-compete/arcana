package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
)

type ProfileStore struct {
	mu       sync.RWMutex
	profiles map[string]FusionProfile
}

type ShardRegistry struct {
	mu     sync.RWMutex
	shards map[string]Shard
}

func NewProfileStore() *ProfileStore {
	profiles := map[string]FusionProfile{
		"default": {
			Name: "default",
			Weights: FusionWeights{
				Semantic:  0.35,
				Temporal:  0.25,
				Authority: 0.25,
				Entity:    0.15,
			},
		},
		"semantic_first": {
			Name: "semantic_first",
			Weights: FusionWeights{
				Semantic:  0.65,
				Temporal:  0.10,
				Authority: 0.15,
				Entity:    0.10,
			},
		},
		"navigation_link": {
			Name: "navigation_link",
			Weights: FusionWeights{
				Semantic:  0.15,
				Temporal:  0.10,
				Authority: 0.35,
				Entity:    0.40,
			},
		},
		"recent_first": {
			Name: "recent_first",
			Weights: FusionWeights{
				Semantic:  0.20,
				Temporal:  0.55,
				Authority: 0.15,
				Entity:    0.10,
			},
		},
	}
	return &ProfileStore{profiles: profiles}
}

func (s *ProfileStore) List() []FusionProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]FusionProfile, 0, len(s.profiles))
	for _, p := range s.profiles {
		out = append(out, p)
	}
	return out
}

func (s *ProfileStore) Get(name string) (FusionProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.profiles[name]
	return p, ok
}

func NewShardRegistry() *ShardRegistry {
	shards := map[string]Shard{
		"shard-platform": {
			ID:           "shard-platform",
			Topic:        "platform-architecture",
			CentroidHash: hashCentroid("kubernetes agents orchestration mesh"),
			DocCount:     1284,
		},
		"shard-codex": {
			ID:           "shard-codex",
			Topic:        "codex-search-ingestion",
			CentroidHash: hashCentroid("semantic search bm25 embedding fusion"),
			DocCount:     892,
		},
		"shard-govern": {
			ID:           "shard-govern",
			Topic:        "governance-guardrails",
			CentroidHash: hashCentroid("ward opa policy guardrails evaluation"),
			DocCount:     567,
		},
		"shard-ops": {
			ID:           "shard-ops",
			Topic:        "operations-deployment",
			CentroidHash: hashCentroid("helm kind temporal postgres redis"),
			DocCount:     743,
		},
	}
	return &ShardRegistry{shards: shards}
}

func (r *ShardRegistry) List() []Shard {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Shard, 0, len(r.shards))
	for _, sh := range r.shards {
		out = append(out, sh)
	}
	return out
}

func (r *ShardRegistry) SelectForQuery(query string, filters map[string]string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if topic, ok := filters["topic"]; ok {
		for id, sh := range r.shards {
			if sh.Topic == topic {
				return []string{id}
			}
		}
	}

	q := strings.ToLower(query)
	selected := make([]string, 0, len(r.shards))
	for id, sh := range r.shards {
		if strings.Contains(q, strings.Split(sh.Topic, "-")[0]) {
			selected = append(selected, id)
		}
	}
	if len(selected) == 0 {
		for id := range r.shards {
			selected = append(selected, id)
		}
	}
	return selected
}

func hashCentroid(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:8])
}

func routeForProfile(profile FusionProfile) string {
	w := profile.Weights
	if w.Semantic >= 0.5 {
		return "semantic"
	}
	if w.Entity >= 0.35 {
		return "entity"
	}
	if w.Temporal >= 0.45 {
		return "hybrid"
	}
	return "hybrid"
}

func buildLocalHits(query string, shardIDs []string, topK int, profile FusionProfile) []SearchHit {
	if topK <= 0 {
		topK = 10
	}
	hits := make([]SearchHit, 0, topK)
	w := profile.Weights
	for i, sid := range shardIDs {
		if len(hits) >= topK {
			break
		}
		score := w.Semantic*0.82 + w.Temporal*0.71 + w.Authority*0.65 + w.Entity*0.58
		score -= float64(i) * 0.03
		hits = append(hits, SearchHit{
			DocID:   fmt.Sprintf("doc-%s-%d", sid, i+1),
			ShardID: sid,
			Score:   score,
			Snippet: fmt.Sprintf("Match for %q in shard %s", query, sid),
			Metadata: map[string]string{
				"profile": profile.Name,
				"source":  "codex-router",
			},
		})
	}
	return hits
}
