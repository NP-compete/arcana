package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type GitopsStore struct {
	mu         sync.RWMutex
	promotions map[string]*Promotion
}

func NewGitopsStore() *GitopsStore {
	return &GitopsStore{promotions: make(map[string]*Promotion)}
}

func defaultGates() []PromotionGate {
	return []PromotionGate{
		{Name: "ward_check", Status: GatePending, Required: true},
		{Name: "probe_eval", Status: GatePending, Required: true},
		{Name: "manual_approval", Status: GatePending, Required: true},
	}
}

func (s *GitopsStore) Create(req CreatePromotionRequest) (*Promotion, error) {
	if req.SourceEnv == "" || req.TargetEnv == "" || req.Agent == "" {
		return nil, fmt.Errorf("source_env, target_env, and agent are required")
	}

	gates := req.Gates
	if len(gates) == 0 {
		gates = defaultGates()
	}
	strategy := req.Strategy
	if strategy == "" {
		strategy = "rolling"
	}

	now := time.Now().UTC()
	promo := &Promotion{
		ID: uuid.New().String(), Source: req.SourceEnv, Target: req.TargetEnv,
		Agent: req.Agent, Status: PromotionPending, Gates: gates,
		Strategy: strategy, ApprovedBy: []string{}, StartedAt: now, UpdatedAt: now,
	}

	s.mu.Lock()
	s.promotions[promo.ID] = promo
	s.mu.Unlock()

	go s.advancePromotion(promo.ID)

	copy := *promo
	return &copy, nil
}

func (s *GitopsStore) advancePromotion(id string) {
	time.Sleep(100 * time.Millisecond)
	s.mu.Lock()
	p, ok := s.promotions[id]
	if !ok {
		s.mu.Unlock()
		return
	}
	p.Status = PromotionInProgress
	p.UpdatedAt = time.Now().UTC()
	autoGateNames := []string{"ward_check", "probe_eval"}
	s.mu.Unlock()

	for _, gateName := range autoGateNames {
		time.Sleep(50 * time.Millisecond)
		s.mu.Lock()
		p, ok = s.promotions[id]
		if !ok {
			s.mu.Unlock()
			return
		}
		for i := range p.Gates {
			if p.Gates[i].Name == gateName {
				p.Gates[i].Status = GatePassed
			}
		}
		p.UpdatedAt = time.Now().UTC()
		s.mu.Unlock()
	}
}

func (s *GitopsStore) List() []Promotion {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Promotion, 0, len(s.promotions))
	for _, p := range s.promotions {
		result = append(result, *p)
	}
	return result
}

func (s *GitopsStore) Get(id string) (*Promotion, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.promotions[id]
	if !ok {
		return nil, false
	}
	copy := *p
	return &copy, true
}

func (s *GitopsStore) Approve(id, gateName, approvedBy string) (*Promotion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.promotions[id]
	if !ok {
		return nil, fmt.Errorf("promotion not found")
	}

	found := false
	for i := range p.Gates {
		if p.Gates[i].Name == gateName {
			p.Gates[i].Status = GateApproved
			p.Gates[i].ApprovedBy = approvedBy
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("gate not found: %s", gateName)
	}

	if approvedBy != "" {
		p.ApprovedBy = append(p.ApprovedBy, approvedBy)
	}

	allPassed := true
	for _, g := range p.Gates {
		if g.Required && g.Status != GatePassed && g.Status != GateApproved {
			allPassed = false
			break
		}
	}
	if allPassed {
		p.Status = PromotionCompleted
	} else {
		p.Status = PromotionApproved
	}
	p.UpdatedAt = time.Now().UTC()
	copy := *p
	return &copy, nil
}

func (s *GitopsStore) Rollback(id string) (*Promotion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.promotions[id]
	if !ok {
		return nil, fmt.Errorf("promotion not found")
	}
	p.Status = PromotionRolledBack
	p.UpdatedAt = time.Now().UTC()
	copy := *p
	return &copy, nil
}
