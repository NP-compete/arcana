package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/NP-compete/arcana/pkg/server"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

var agentGVR = schema.GroupVersionResource{
	Group:    "arcana.io",
	Version:  "v1alpha1",
	Resource: "arcanaagents",
}

type OperatorStatus struct {
	Healthy        bool      `json:"healthy"`
	LastReconcile  time.Time `json:"last_reconcile"`
	AgentsManaged  int       `json:"agents_managed"`
	ReconcileCount int64     `json:"reconcile_count"`
	Errors         []string  `json:"errors,omitempty"`
}

var status = OperatorStatus{Healthy: true}

func main() {
	httpSrv := server.New(server.Config{
		ServiceName: "operator",
		Port:        "8082",
	})

	httpSrv.HandleFunc("/status", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	})

	go reconcileLoop()

	httpSrv.ListenAndServe()
}

func reconcileLoop() {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("operator: not in cluster, running in dry-run mode: %v", err)
		for {
			status.LastReconcile = time.Now()
			status.ReconcileCount++
			time.Sleep(30 * time.Second)
		}
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Printf("operator: failed to create dynamic client: %v", err)
		status.Healthy = false
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

	agents, err := client.Resource(agentGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("operator: list agents failed: %v", err)
		status.Errors = append(status.Errors, fmt.Sprintf("%s: %v", time.Now().Format(time.RFC3339), err))
		if len(status.Errors) > 10 {
			status.Errors = status.Errors[len(status.Errors)-10:]
		}
		return
	}

	status.AgentsManaged = len(agents.Items)
	status.LastReconcile = time.Now()
	status.ReconcileCount++
	status.Errors = nil

	for _, agent := range agents.Items {
		reconcileAgent(ctx, client, &agent)
	}

	log.Printf("operator: reconciled %d agents (cycle %d)", len(agents.Items), status.ReconcileCount)
}

func reconcileAgent(ctx context.Context, client dynamic.Interface, agent *unstructured.Unstructured) {
	name := agent.GetName()
	namespace := agent.GetNamespace()

	currentStatus, found, _ := unstructured.NestedString(agent.Object, "status", "phase")
	if !found || currentStatus == "" {
		patch := []byte(`{"status":{"phase":"Provisioned","message":"Reconciled by operator","lastUpdated":"` + time.Now().Format(time.RFC3339) + `"}}`)
		_, err := client.Resource(agentGVR).Namespace(namespace).Patch(ctx, name, "application/merge-patch+json", patch, metav1.PatchOptions{}, "status")
		if err != nil {
			log.Printf("operator: failed to patch status for %s/%s: %v", namespace, name, err)
		} else {
			log.Printf("operator: set status Provisioned for %s/%s", namespace, name)
		}
	}
}
