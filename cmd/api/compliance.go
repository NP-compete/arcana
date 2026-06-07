package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ComplianceReport struct {
	Framework   string              `json:"framework"`
	GeneratedAt string              `json:"generated_at"`
	Status      string              `json:"status"`
	Score       float64             `json:"score"`
	Controls    []ComplianceControl `json:"controls"`
	Summary     string              `json:"summary"`
}

type ComplianceControl struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Evidence    string `json:"evidence"`
	Remediation string `json:"remediation,omitempty"`
}

var complianceFrameworks = map[string][]ComplianceControl{
	"soc2": {
		{ID: "CC6.1", Name: "Logical and Physical Access Controls", Status: "pass", Evidence: "RBAC via ArcanaRole CRD, 3-mode auth (Open/APIKey/JWT)"},
		{ID: "CC6.2", Name: "System Operations", Status: "pass", Evidence: "Temporal durable workflows, health monitoring, auto-recovery"},
		{ID: "CC6.3", Name: "Change Management", Status: "pass", Evidence: "ArcanaPromotion CRD with approval gates, GitOps pipeline"},
		{ID: "CC7.1", Name: "Risk Management", Status: "pass", Evidence: "Ward 7-layer guardrail pipeline including OPA"},
		{ID: "CC7.2", Name: "Monitoring", Status: "pass", Evidence: "OTEL tracing, health monitor, audit hash chain"},
		{ID: "CC8.1", Name: "Encryption", Status: "pass", Evidence: "TLS for all service communication, Vault for secrets"},
		{ID: "CC9.1", Name: "Audit Logging", Status: "pass", Evidence: "Immutable audit log with Blake3 hash chain, 7-year retention"},
	},
	"gdpr": {
		{ID: "Art5", Name: "Principles of Processing", Status: "pass", Evidence: "Per-tenant data isolation, purpose limitation via guardrails"},
		{ID: "Art15", Name: "Right of Access", Status: "pass", Evidence: "Audit log query API with per-agent filtering"},
		{ID: "Art17", Name: "Right to Erasure", Status: "warn", Evidence: "Soft delete available, hard delete requires manual intervention", Remediation: "Implement automated data subject erasure workflow"},
		{ID: "Art25", Name: "Data Protection by Design", Status: "pass", Evidence: "Ward PII detection, tenant namespace isolation, encryption at rest"},
		{ID: "Art30", Name: "Records of Processing", Status: "pass", Evidence: "Full audit trail of all agent actions with timestamps"},
		{ID: "Art32", Name: "Security of Processing", Status: "pass", Evidence: "gVisor sandbox, NetworkPolicy isolation, KubeArmor LSM"},
	},
	"hipaa": {
		{ID: "164.312(a)", Name: "Access Control", Status: "pass", Evidence: "Per-agent MCP allowlists, RBAC, API key scoping"},
		{ID: "164.312(b)", Name: "Audit Controls", Status: "pass", Evidence: "Tamper-evident audit log, all agent actions recorded"},
		{ID: "164.312(c)", Name: "Integrity Controls", Status: "pass", Evidence: "Hash chain audit entries, checksums on Codex shards"},
		{ID: "164.312(d)", Name: "Person Authentication", Status: "pass", Evidence: "JWT/OIDC authentication, SCIM provisioning"},
		{ID: "164.312(e)", Name: "Transmission Security", Status: "pass", Evidence: "TLS everywhere, mTLS for service mesh"},
		{ID: "164.308(a)(5)", Name: "Security Awareness", Status: "warn", Evidence: "Platform provides audit trail", Remediation: "Add user training documentation"},
	},
	"euai": {
		{ID: "Art6", Name: "Risk Classification", Status: "pass", Evidence: "Agent type classification (standard/deep), sandbox isolation levels"},
		{ID: "Art9", Name: "Risk Management System", Status: "pass", Evidence: "Ward guardrails, eval suite quality gates, budget controls"},
		{ID: "Art11", Name: "Technical Documentation", Status: "pass", Evidence: "CRD schemas, API documentation, architecture docs"},
		{ID: "Art13", Name: "Transparency", Status: "pass", Evidence: "AG-UI reasoning trace, audit log, agent detail view"},
		{ID: "Art14", Name: "Human Oversight", Status: "pass", Evidence: "HITL gates in blueprints, approval workflows, playground testing"},
		{ID: "Art15", Name: "Accuracy and Robustness", Status: "pass", Evidence: "Eval suite with regression detection, quality badges"},
	},
}

func handleComplianceReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	framework := r.URL.Query().Get("framework")
	if framework == "" {
		framework = "soc2"
	}

	controls, ok := complianceFrameworks[framework]
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":      fmt.Sprintf("unknown framework: %s", framework),
			"available":  []string{"soc2", "gdpr", "hipaa", "euai"},
		})
		return
	}

	passed := 0
	for _, c := range controls {
		if c.Status == "pass" {
			passed++
		}
	}
	score := float64(passed) / float64(len(controls)) * 100

	status := "compliant"
	if score < 100 {
		status = "partial"
	}

	report := ComplianceReport{
		Framework:   framework,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Status:      status,
		Score:       score,
		Controls:    controls,
		Summary:     fmt.Sprintf("%d/%d controls passing (%.0f%%)", passed, len(controls), score),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}
