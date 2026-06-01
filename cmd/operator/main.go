package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/NP-compete/arcana/pkg/logger"
	"github.com/NP-compete/arcana/pkg/server"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

const (
	finalizerName    = "arcana.io/agent-cleanup"
	reconcileTimeout = 30 * time.Second
	maxRetryBackoff  = 5 // max consecutive failures before capping backoff exponent
)

// GVRs for Kubernetes resources managed by the operator.
var (
	agentGVR = schema.GroupVersionResource{
		Group:    "arcana.io",
		Version:  "v1alpha1",
		Resource: "arcanaagents",
	}

	namespaceGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "namespaces",
	}

	configMapGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "configmaps",
	}

	networkPolicyGVR = schema.GroupVersionResource{
		Group:    "networking.k8s.io",
		Version:  "v1",
		Resource: "networkpolicies",
	}

	resourceQuotaGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "resourcequotas",
	}

	deploymentGVR = schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
)

// OperatorStatus exposes operator health via the /status endpoint.
type OperatorStatus struct {
	Healthy        bool      `json:"healthy"`
	LastReconcile  time.Time `json:"last_reconcile"`
	AgentsManaged  int       `json:"agents_managed"`
	ReconcileCount int64     `json:"reconcile_count"`
	Errors         []string  `json:"errors,omitempty"`
}

var (
	status   = OperatorStatus{Healthy: true}
	statusMu sync.RWMutex
)

// retryTracker holds per-agent consecutive failure counts for exponential backoff.
type retryTracker struct {
	mu     sync.Mutex
	counts map[string]int
}

func newRetryTracker() *retryTracker {
	return &retryTracker{counts: make(map[string]int)}
}

func (rt *retryTracker) recordSuccess(key string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	delete(rt.counts, key)
}

func (rt *retryTracker) recordFailure(key string) int {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.counts[key]++
	return rt.counts[key]
}

// shouldSkip returns true if the agent should be skipped this cycle due to backoff.
// Backoff: skip unless the current cycle is aligned to a 2^(failures) interval,
// capped at 2^maxRetryBackoff.
func (rt *retryTracker) shouldSkip(key string, cycle int64) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	failures := rt.counts[key]
	if failures == 0 {
		return false
	}
	capped := failures
	if capped > maxRetryBackoff {
		capped = maxRetryBackoff
	}
	interval := int64(1) << capped
	return (cycle % interval) != 0
}

func main() {
	log := logger.New("operator")

	httpSrv := server.New(server.Config{
		ServiceName: "operator",
		Port:        "8082",
	})

	httpSrv.HandleFunc("/status", func(w http.ResponseWriter, _ *http.Request) {
		statusMu.RLock()
		defer statusMu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	})

	go reconcileLoop(log)

	if err := httpSrv.ListenAndServe(); err != nil {
		log.Error("server exited with error", "error", err.Error())
	}
}

func reconcileLoop(log *logger.Logger) {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Warn("not in cluster, running in dry-run mode", "error", err.Error())
		for {
			statusMu.Lock()
			status.LastReconcile = time.Now()
			status.ReconcileCount++
			statusMu.Unlock()
			time.Sleep(30 * time.Second)
		}
	}

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Error("failed to create dynamic client", "error", err.Error())
		statusMu.Lock()
		status.Healthy = false
		statusMu.Unlock()
		return
	}

	watchNamespace := os.Getenv("WATCH_NAMESPACE")
	if watchNamespace == "" {
		log.Info("WATCH_NAMESPACE not set, watching all namespaces")
	} else {
		log.Info("watching single namespace", "namespace", watchNamespace)
	}

	tracker := newRetryTracker()

	log.Info("starting reconcile loop", "interval", "30s")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	reconcile(dynClient, watchNamespace, tracker, log)

	for range ticker.C {
		reconcile(dynClient, watchNamespace, tracker, log)
	}
}

func reconcile(dynClient dynamic.Interface, watchNamespace string, tracker *retryTracker, log *logger.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	defer cancel()

	var agents *unstructured.UnstructuredList
	var err error
	if watchNamespace != "" {
		agents, err = dynClient.Resource(agentGVR).Namespace(watchNamespace).List(ctx, metav1.ListOptions{})
	} else {
		agents, err = dynClient.Resource(agentGVR).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		log.Error("failed to list agents", "error", err.Error())
		statusMu.Lock()
		status.Errors = appendCapped(status.Errors, fmt.Sprintf("%s: list agents: %v", time.Now().Format(time.RFC3339), err), 10)
		statusMu.Unlock()
		return
	}

	statusMu.Lock()
	status.AgentsManaged = len(agents.Items)
	status.LastReconcile = time.Now()
	status.ReconcileCount++
	cycle := status.ReconcileCount
	status.Errors = nil
	statusMu.Unlock()

	for i := range agents.Items {
		agent := &agents.Items[i]
		agentKey := fmt.Sprintf("%s/%s", agent.GetNamespace(), agent.GetName())

		if tracker.shouldSkip(agentKey, cycle) {
			log.Debug("skipping agent due to backoff", "agent", agentKey)
			continue
		}

		if reconcileErr := reconcileAgent(ctx, dynClient, agent, log); reconcileErr != nil {
			failures := tracker.recordFailure(agentKey)
			log.Error("reconcile failed for agent",
				"agent", agentKey,
				"error", reconcileErr.Error(),
				"consecutive_failures", failures,
			)
			statusMu.Lock()
			status.Errors = appendCapped(status.Errors, fmt.Sprintf("%s: %s: %v", time.Now().Format(time.RFC3339), agentKey, reconcileErr), 10)
			statusMu.Unlock()
		} else {
			tracker.recordSuccess(agentKey)
		}
	}

	log.Info("reconcile cycle complete", "agents", len(agents.Items), "cycle", cycle)
}

func reconcileAgent(ctx context.Context, dynClient dynamic.Interface, agent *unstructured.Unstructured, log *logger.Logger) error {
	name := agent.GetName()
	namespace := agent.GetNamespace()
	agentKey := fmt.Sprintf("%s/%s", namespace, name)

	// Handle deletion: if deletionTimestamp is set, run cleanup and remove finalizer.
	if agent.GetDeletionTimestamp() != nil {
		log.Info("agent marked for deletion, running cleanup", "agent", agentKey)
		if err := cleanupAgentResources(ctx, dynClient, agent, log); err != nil {
			setStatusCondition(ctx, dynClient, agent, "Error", "False", "CleanupFailed",
				fmt.Sprintf("Failed to clean up resources: %v", err), log)
			return fmt.Errorf("cleanup failed: %w", err)
		}
		if err := removeFinalizer(ctx, dynClient, agent); err != nil {
			return fmt.Errorf("remove finalizer: %w", err)
		}
		log.Info("cleanup complete, finalizer removed", "agent", agentKey)
		return nil
	}

	// Ensure finalizer is present on first reconcile.
	if !hasFinalizer(agent, finalizerName) {
		if err := addFinalizer(ctx, dynClient, agent); err != nil {
			return fmt.Errorf("add finalizer: %w", err)
		}
		log.Info("finalizer added", "agent", agentKey)
	}

	agentNS := agentNamespace(name)
	var degraded []string

	// Step 1: Ensure agent namespace exists.
	if err := ensureNamespace(ctx, dynClient, agentNS, map[string]string{
		"app.kubernetes.io/part-of":    "arcana",
		"app.kubernetes.io/managed-by": "arcana-operator",
		"arcana.io/agent":              name,
	}); err != nil {
		setStatusCondition(ctx, dynClient, agent, "Error", "False", "NamespaceCreationFailed",
			fmt.Sprintf("Failed to create namespace %s: %v", agentNS, err), log)
		return fmt.Errorf("ensure namespace %s: %w", agentNS, err)
	}

	// Step 2: Ensure agent ConfigMap exists with the agent's spec.
	configData := buildConfigMapData(agent)
	if err := ensureConfigMap(ctx, dynClient, agentNS, "agent-config", configData); err != nil {
		degraded = append(degraded, fmt.Sprintf("configmap: %v", err))
		log.Warn("failed to ensure configmap", "agent", agentKey, "error", err.Error())
	}

	// Step 3: Ensure NetworkPolicy exists.
	if err := ensureNetworkPolicy(ctx, dynClient, agentNS, "agent-network-policy"); err != nil {
		degraded = append(degraded, fmt.Sprintf("networkpolicy: %v", err))
		log.Warn("failed to ensure network policy", "agent", agentKey, "error", err.Error())
	}

	// Step 4: Ensure ResourceQuota exists.
	if err := ensureResourceQuota(ctx, dynClient, agentNS, "agent-quota"); err != nil {
		degraded = append(degraded, fmt.Sprintf("resourcequota: %v", err))
		log.Warn("failed to ensure resource quota", "agent", agentKey, "error", err.Error())
	}

	// Step 5: For deep agents, check if a Deployment exists.
	agentType, _, _ := unstructured.NestedString(agent.Object, "spec", "agentType")
	if agentType == "deep" {
		exists, err := deploymentExists(ctx, dynClient, agentNS, "deep-agent")
		if err != nil {
			degraded = append(degraded, fmt.Sprintf("deployment check: %v", err))
			log.Warn("failed to check deep agent deployment", "agent", agentKey, "error", err.Error())
		} else if !exists {
			log.Info("deep agent deployment not found, provisioning needed",
				"agent", agentKey, "namespace", agentNS)
			degraded = append(degraded, "deployment: not yet provisioned")
		}
	}

	// Step 6: Set status condition.
	if len(degraded) > 0 {
		msg := fmt.Sprintf("Some resources are not ready: %v", degraded)
		setStatusCondition(ctx, dynClient, agent, "Degraded", "False", "ResourcesNotReady", msg, log)
	} else {
		setStatusCondition(ctx, dynClient, agent, "Ready", "True", "AllResourcesReady",
			"All agent resources are provisioned and healthy", log)
	}

	return nil
}

// --- Namespace operations ---

func agentNamespace(agentName string) string {
	return "arcana-agent-" + agentName
}

func ensureNamespace(ctx context.Context, dynClient dynamic.Interface, name string, labels map[string]string) error {
	_, err := dynClient.Resource(namespaceGVR).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return nil
	}

	ns := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name":   name,
				"labels": toStringInterfaceMap(labels),
			},
		},
	}

	_, err = dynClient.Resource(namespaceGVR).Create(ctx, ns, metav1.CreateOptions{})
	if err != nil && !isAlreadyExists(err) {
		return fmt.Errorf("create namespace: %w", err)
	}
	return nil
}

func deleteNamespace(ctx context.Context, dynClient dynamic.Interface, name string) error {
	err := dynClient.Resource(namespaceGVR).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("delete namespace %s: %w", name, err)
	}
	return nil
}

// --- ConfigMap operations ---

func ensureConfigMap(ctx context.Context, dynClient dynamic.Interface, namespace, name string, data map[string]string) error {
	_, err := dynClient.Resource(configMapGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return nil
	}

	cm := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"app.kubernetes.io/part-of":    "arcana",
					"app.kubernetes.io/managed-by": "arcana-operator",
				},
			},
			"data": toStringInterfaceMap(data),
		},
	}

	_, err = dynClient.Resource(configMapGVR).Namespace(namespace).Create(ctx, cm, metav1.CreateOptions{})
	if err != nil && !isAlreadyExists(err) {
		return fmt.Errorf("create configmap %s/%s: %w", namespace, name, err)
	}
	return nil
}

func buildConfigMapData(agent *unstructured.Unstructured) map[string]string {
	data := map[string]string{
		"agent-name": agent.GetName(),
	}

	if model, found, _ := unstructured.NestedString(agent.Object, "spec", "model"); found {
		data["model"] = model
	}

	if skills, found, _ := unstructured.NestedStringSlice(agent.Object, "spec", "skills"); found {
		for i, s := range skills {
			data[fmt.Sprintf("skill-%d", i)] = s
		}
	}

	if backend, found, _ := unstructured.NestedString(agent.Object, "spec", "memory", "backend"); found {
		data["memory-backend"] = backend
	}
	if ttl, found, _ := unstructured.NestedString(agent.Object, "spec", "memory", "ttl"); found {
		data["memory-ttl"] = ttl
	}

	if strategy, found, _ := unstructured.NestedString(agent.Object, "spec", "budget", "routingStrategy"); found {
		data["routing-strategy"] = strategy
	}
	if maxTokens, found, _ := unstructured.NestedFieldNoCopy(agent.Object, "spec", "budget", "maxTokensPerTurn"); found {
		data["max-tokens-per-turn"] = fmt.Sprintf("%v", maxTokens)
	}

	if runtime, found, _ := unstructured.NestedString(agent.Object, "spec", "sandbox", "runtime"); found {
		data["sandbox-runtime"] = runtime
	}

	return data
}

// --- NetworkPolicy operations ---

func ensureNetworkPolicy(ctx context.Context, dynClient dynamic.Interface, namespace, name string) error {
	_, err := dynClient.Resource(networkPolicyGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return nil
	}

	np := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.k8s.io/v1",
			"kind":       "NetworkPolicy",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"app.kubernetes.io/part-of":    "arcana",
					"app.kubernetes.io/managed-by": "arcana-operator",
				},
			},
			"spec": map[string]interface{}{
				"podSelector": map[string]interface{}{},
				"policyTypes": []interface{}{"Ingress", "Egress"},
				"ingress": []interface{}{
					map[string]interface{}{
						"from": []interface{}{
							map[string]interface{}{
								"namespaceSelector": map[string]interface{}{
									"matchLabels": map[string]interface{}{
										"app.kubernetes.io/part-of": "arcana",
									},
								},
							},
						},
					},
				},
				"egress": []interface{}{
					map[string]interface{}{
						"to": []interface{}{
							map[string]interface{}{
								"namespaceSelector": map[string]interface{}{
									"matchLabels": map[string]interface{}{
										"app.kubernetes.io/part-of": "arcana",
									},
								},
							},
						},
					},
					map[string]interface{}{
						"to": []interface{}{
							map[string]interface{}{
								"namespaceSelector": map[string]interface{}{},
							},
						},
						"ports": []interface{}{
							map[string]interface{}{"protocol": "TCP", "port": int64(53)},
							map[string]interface{}{"protocol": "UDP", "port": int64(53)},
						},
					},
				},
			},
		},
	}

	_, err = dynClient.Resource(networkPolicyGVR).Namespace(namespace).Create(ctx, np, metav1.CreateOptions{})
	if err != nil && !isAlreadyExists(err) {
		return fmt.Errorf("create networkpolicy %s/%s: %w", namespace, name, err)
	}
	return nil
}

// --- ResourceQuota operations ---

func ensureResourceQuota(ctx context.Context, dynClient dynamic.Interface, namespace, name string) error {
	_, err := dynClient.Resource(resourceQuotaGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return nil
	}

	rq := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ResourceQuota",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"app.kubernetes.io/part-of":    "arcana",
					"app.kubernetes.io/managed-by": "arcana-operator",
				},
			},
			"spec": map[string]interface{}{
				"hard": map[string]interface{}{
					"pods":                   "20",
					"requests.cpu":           "4",
					"requests.memory":        "8Gi",
					"limits.cpu":             "8",
					"limits.memory":          "16Gi",
					"persistentvolumeclaims": "5",
				},
			},
		},
	}

	_, err = dynClient.Resource(resourceQuotaGVR).Namespace(namespace).Create(ctx, rq, metav1.CreateOptions{})
	if err != nil && !isAlreadyExists(err) {
		return fmt.Errorf("create resourcequota %s/%s: %w", namespace, name, err)
	}
	return nil
}

// --- Deployment check ---

func deploymentExists(ctx context.Context, dynClient dynamic.Interface, namespace, name string) (bool, error) {
	_, err := dynClient.Resource(deploymentGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return true, nil
	}
	if isNotFound(err) {
		return false, nil
	}
	return false, fmt.Errorf("get deployment %s/%s: %w", namespace, name, err)
}

// --- Finalizer operations ---

func hasFinalizer(obj *unstructured.Unstructured, finalizer string) bool {
	for _, f := range obj.GetFinalizers() {
		if f == finalizer {
			return true
		}
	}
	return false
}

func addFinalizer(ctx context.Context, dynClient dynamic.Interface, agent *unstructured.Unstructured) error {
	finalizers := agent.GetFinalizers()
	finalizers = append(finalizers, finalizerName)

	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"finalizers": finalizers,
		},
	}
	patchData, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshal finalizer patch: %w", err)
	}

	_, err = dynClient.Resource(agentGVR).Namespace(agent.GetNamespace()).Patch(
		ctx, agent.GetName(), types.MergePatchType, patchData, metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("patch finalizer: %w", err)
	}

	// Update local copy so subsequent checks in this cycle see the finalizer.
	agent.SetFinalizers(finalizers)
	return nil
}

func removeFinalizer(ctx context.Context, dynClient dynamic.Interface, agent *unstructured.Unstructured) error {
	var updated []string
	for _, f := range agent.GetFinalizers() {
		if f != finalizerName {
			updated = append(updated, f)
		}
	}

	// Use a JSON merge patch. An empty slice must serialize as [] not null.
	finalizersValue := interface{}(updated)
	if len(updated) == 0 {
		finalizersValue = []interface{}{}
	}

	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"finalizers": finalizersValue,
		},
	}
	patchData, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshal finalizer removal patch: %w", err)
	}

	_, err = dynClient.Resource(agentGVR).Namespace(agent.GetNamespace()).Patch(
		ctx, agent.GetName(), types.MergePatchType, patchData, metav1.PatchOptions{},
	)
	return err
}

// --- Cleanup ---

func cleanupAgentResources(ctx context.Context, dynClient dynamic.Interface, agent *unstructured.Unstructured, log *logger.Logger) error {
	agentNS := agentNamespace(agent.GetName())

	// Check if namespace exists before attempting deletion.
	_, err := dynClient.Resource(namespaceGVR).Get(ctx, agentNS, metav1.GetOptions{})
	if err != nil {
		if isNotFound(err) {
			log.Info("agent namespace already removed", "namespace", agentNS)
			return nil
		}
		return fmt.Errorf("check namespace %s: %w", agentNS, err)
	}

	// Delete the namespace -- Kubernetes cascades deletion of all namespaced resources.
	if err := deleteNamespace(ctx, dynClient, agentNS); err != nil {
		return err
	}
	log.Info("agent namespace deletion initiated", "namespace", agentNS)
	return nil
}

// --- Status condition ---

func setStatusCondition(ctx context.Context, dynClient dynamic.Interface, agent *unstructured.Unstructured, condType, condStatus, reason, message string, log *logger.Logger) {
	now := time.Now().UTC().Format(time.RFC3339)

	condition := map[string]interface{}{
		"type":               condType,
		"status":             condStatus,
		"lastTransitionTime": now,
		"reason":             reason,
		"message":            message,
	}

	patch := map[string]interface{}{
		"status": map[string]interface{}{
			"phase":      condType,
			"message":    message,
			"conditions": []interface{}{condition},
		},
	}

	patchData, err := json.Marshal(patch)
	if err != nil {
		log.Error("failed to marshal status patch",
			"agent", fmt.Sprintf("%s/%s", agent.GetNamespace(), agent.GetName()),
			"error", err.Error(),
		)
		return
	}

	_, err = dynClient.Resource(agentGVR).Namespace(agent.GetNamespace()).Patch(
		ctx, agent.GetName(), types.MergePatchType, patchData, metav1.PatchOptions{}, "status",
	)
	if err != nil {
		log.Error("failed to patch agent status",
			"agent", fmt.Sprintf("%s/%s", agent.GetNamespace(), agent.GetName()),
			"error", err.Error(),
		)
	}
}

// --- Error classification ---

func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "already exists")
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "not found")
}

func contains(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- Utility ---

func appendCapped(slice []string, item string, max int) []string {
	slice = append(slice, item)
	if len(slice) > max {
		slice = slice[len(slice)-max:]
	}
	return slice
}

func toStringInterfaceMap(m map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
