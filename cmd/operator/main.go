package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/NP-compete/arcana/pkg/server"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

var agentGVR = schema.GroupVersionResource{
	Group:    "arcana.ai",
	Version:  "v1alpha1",
	Resource: "arcanaagents",
}

type OperatorStatus struct {
	Healthy        bool      `json:"healthy"`
	LastReconcile  time.Time `json:"last_reconcile"`
	AgentsManaged  int       `json:"agents_managed"`
	ReconcileCount int64     `json:"reconcile_count"`
	Provisioned    int       `json:"provisioned"`
	Failed         int       `json:"failed"`
	Errors         []string  `json:"errors,omitempty"`
}

var opStatus = OperatorStatus{Healthy: true}

var meshHost string
var httpClient = &http.Client{Timeout: 10 * time.Second}

func main() {
	meshHost = os.Getenv("MESH_HOST")
	if meshHost == "" {
		meshHost = "arcana-mesh.arcana.svc.cluster.local"
	}

	httpSrv := server.New(server.Config{
		ServiceName: "operator",
		Port:        "8082",
	})

	httpSrv.HandleFunc("/status", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(opStatus)
	})

	go reconcileLoop()

	httpSrv.ListenAndServe()
}

func reconcileLoop() {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("operator: not in cluster, running in dry-run mode: %v", err)
		for {
			opStatus.LastReconcile = time.Now()
			opStatus.ReconcileCount++
			time.Sleep(30 * time.Second)
		}
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Printf("operator: failed to create dynamic client: %v", err)
		opStatus.Healthy = false
		return
	}

	log.Println("operator: starting reconcile loop (30s interval)")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	reconcile(client)

	for range ticker.C {
		reconcile(client)
	}
}

func reconcile(client dynamic.Interface) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reconcileTenants(ctx, client)
	reconcileBackupPolicies(ctx, client)

	agents, err := client.Resource(agentGVR).Namespace("arcana").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("operator: list agents failed: %v", err)
		appendError(fmt.Sprintf("%s: %v", time.Now().Format(time.RFC3339), err))
		return
	}

	opStatus.AgentsManaged = len(agents.Items)
	opStatus.LastReconcile = time.Now()
	opStatus.ReconcileCount++
	opStatus.Errors = nil

	provisioned := 0
	failed := 0
	for _, agent := range agents.Items {
		if reconcileAgent(ctx, client, &agent) {
			provisioned++
		} else {
			failed++
		}
	}
	opStatus.Provisioned = provisioned
	opStatus.Failed = failed

	log.Printf("operator: reconciled %d agents (%d provisioned, %d failed, cycle %d)",
		len(agents.Items), provisioned, failed, opStatus.ReconcileCount)
}

func reconcileAgent(ctx context.Context, client dynamic.Interface, agent *unstructured.Unstructured) bool {
	name := agent.GetName()
	namespace := agent.GetNamespace()

	currentPhase, _, _ := unstructured.NestedString(agent.Object, "status", "phase")

	spec, _ := agent.Object["spec"].(map[string]interface{})
	if spec == nil {
		spec = map[string]interface{}{}
	}

	switch currentPhase {
	case "":
		log.Printf("operator: new agent %s/%s — registering with mesh", namespace, name)
		if err := registerWithMesh(name, spec); err != nil {
			log.Printf("operator: mesh registration failed for %s: %v", name, err)
			patchPhase(ctx, client, namespace, name, "Failed", fmt.Sprintf("mesh registration: %v", err))
			return false
		}
		patchPhase(ctx, client, namespace, name, "Provisioned", "Registered with mesh")
		return true

	case "Provisioned":
		return true

	case "Failed":
		log.Printf("operator: agent %s/%s in Failed state, retrying", namespace, name)
		if err := registerWithMesh(name, spec); err != nil {
			return false
		}
		patchPhase(ctx, client, namespace, name, "Provisioned", "Recovered from failed state")
		return true

	default:
		return true
	}
}

func registerWithMesh(name string, spec map[string]interface{}) error {
	model, _ := spec["model"].(string)
	skills, _ := toStringSlice(spec["skills"])

	capabilities := skills
	if model != "" {
		capabilities = append(capabilities, model)
	}

	agentType := "create_agent"
	var deepConfig map[string]interface{}

	if sandbox, ok := spec["sandbox"].(map[string]interface{}); ok {
		if _, hasRuntime := sandbox["runtime"]; hasRuntime {
			agentType = "create_deep_agent"
			deepConfig = map[string]interface{}{
				"world_model":  false,
				"skill_graph":  len(skills) > 0,
				"hitl_enabled": false,
				"self_improve": false,
				"temperature":  0.7,
			}
		}
	}

	if memory, ok := spec["memory"].(map[string]interface{}); ok {
		if deepConfig == nil {
			agentType = "create_deep_agent"
			deepConfig = map[string]interface{}{}
		}
		if policy, ok := memory["backend"].(string); ok {
			deepConfig["memory_policy"] = policy
		}
	}

	body := map[string]interface{}{
		"name":         name,
		"agent_type":   agentType,
		"capabilities": capabilities,
		"protocols":    []string{"mcp", "a2a"},
	}
	if deepConfig != nil {
		body["deep_config"] = deepConfig
	}

	payload, _ := json.Marshal(body)
	resp, err := httpClient.Post(
		fmt.Sprintf("http://%s:8083/api/v1/agents/register", meshHost),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("mesh unreachable: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		return fmt.Errorf("mesh returned %d: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("operator: registered agent %s with mesh (type=%s)", name, agentType)
	return nil
}

func patchPhase(ctx context.Context, client dynamic.Interface, namespace, name, phase, message string) {
	patch := []byte(fmt.Sprintf(
		`{"status":{"phase":"%s","message":"%s","lastUpdated":"%s"}}`,
		phase, message, time.Now().Format(time.RFC3339),
	))
	_, err := client.Resource(agentGVR).Namespace(namespace).Patch(
		ctx, name, "application/merge-patch+json", patch, metav1.PatchOptions{}, "status",
	)
	if err != nil {
		log.Printf("operator: patch status %s for %s/%s: %v", phase, namespace, name, err)
	}
}

func toStringSlice(v interface{}) ([]string, bool) {
	arr, ok := v.([]interface{})
	if !ok {
		return nil, false
	}
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result, true
}

func appendError(msg string) {
	opStatus.Errors = append(opStatus.Errors, msg)
	if len(opStatus.Errors) > 10 {
		opStatus.Errors = opStatus.Errors[len(opStatus.Errors)-10:]
	}
}
