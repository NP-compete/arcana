package main

import "time"

type PromotionStatus string

const (
	PromotionPending    PromotionStatus = "pending"
	PromotionInProgress PromotionStatus = "in_progress"
	PromotionApproved   PromotionStatus = "approved"
	PromotionCompleted  PromotionStatus = "completed"
	PromotionFailed     PromotionStatus = "failed"
	PromotionRolledBack PromotionStatus = "rolled_back"
)

type GateStatus string

const (
	GatePending  GateStatus = "pending"
	GatePassed   GateStatus = "passed"
	GateFailed   GateStatus = "failed"
	GateApproved GateStatus = "approved"
)

type PromotionGate struct {
	Name       string     `json:"name"`
	Status     GateStatus `json:"status"`
	Required   bool       `json:"required"`
	ApprovedBy string     `json:"approved_by,omitempty"`
}

type Promotion struct {
	ID         string          `json:"id"`
	Source     string          `json:"source"`
	Target     string          `json:"target"`
	Agent      string          `json:"agent"`
	Status     PromotionStatus `json:"status"`
	Gates      []PromotionGate `json:"gates"`
	Strategy   string          `json:"strategy"`
	ApprovedBy []string        `json:"approved_by"`
	StartedAt  time.Time       `json:"started_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type CreatePromotionRequest struct {
	SourceEnv  string          `json:"source_env"`
	TargetEnv  string          `json:"target_env"`
	Agent      string          `json:"agent"`
	Gates      []PromotionGate `json:"gates"`
	Strategy   string          `json:"strategy"`
}

type ApproveRequest struct {
	GateName   string `json:"gate_name"`
	ApprovedBy string `json:"approved_by"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
