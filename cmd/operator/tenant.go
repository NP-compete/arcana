package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var tenantGVR = schema.GroupVersionResource{
	Group:    "arcana.io",
	Version:  "v1alpha1",
	Resource: "arcanatenants",
}

func reconcileTenants(ctx context.Context, client dynamic.Interface) {
	tenants, err := client.Resource(tenantGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("operator: list tenants failed: %v", err)
		return
	}

	for _, tenant := range tenants.Items {
		reconcileTenant(ctx, client, &tenant)
	}

	log.Printf("operator: reconciled %d tenants", len(tenants.Items))
}

func reconcileTenant(ctx context.Context, client dynamic.Interface, tenant *unstructured.Unstructured) {
	name := tenant.GetName()
	spec, _ := tenant.Object["spec"].(map[string]interface{})

	currentPhase, _, _ := unstructured.NestedString(tenant.Object, "status", "phase")

	if currentPhase == "Active" {
		return
	}

	displayName, _ := spec["displayName"].(string)
	if displayName == "" {
		displayName = name
	}

	namespaces, _ := toStringSlice(spec["namespaces"])
	if len(namespaces) == 0 {
		namespaces = []string{"arcana-tenant-" + name}
	}

	quotas, _ := spec["quotas"].(map[string]interface{})
	maxAgents := 20
	if q, ok := quotas["maxAgents"].(float64); ok {
		maxAgents = int(q)
	}

	for _, ns := range namespaces {
		if err := createTenantNamespace(ctx, client, ns, name, maxAgents); err != nil {
			log.Printf("operator: failed to create namespace %s for tenant %s: %v", ns, name, err)
			patchTenantPhase(ctx, client, name, "Failed", fmt.Sprintf("namespace %s: %v", ns, err))
			return
		}
	}

	if err := registerTenantInMesh(name, displayName, namespaces, maxAgents); err != nil {
		log.Printf("operator: failed to register tenant %s in mesh: %v", name, err)
	}

	patchTenantPhase(ctx, client, name, "Active", "Tenant provisioned")
	log.Printf("operator: tenant %s provisioned (%d namespaces, maxAgents=%d)", name, len(namespaces), maxAgents)
}

func createTenantNamespace(ctx context.Context, client dynamic.Interface, namespace, tenantName string, maxAgents int) error {
	nsGVR := schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}

	_, err := client.Resource(nsGVR).Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		return nil
	}

	ns := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": namespace,
				"labels": map[string]interface{}{
					"arcana.io/tenant":     tenantName,
					"arcana.io/managed-by": "arcana-operator",
				},
			},
		},
	}

	_, err = client.Resource(nsGVR).Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create namespace: %w", err)
	}

	rqGVR := schema.GroupVersionResource{Version: "v1", Resource: "resourcequotas"}
	rq := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ResourceQuota",
			"metadata": map[string]interface{}{
				"name":      "tenant-quota",
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"hard": map[string]interface{}{
					"pods":            fmt.Sprintf("%d", maxAgents*2),
					"requests.cpu":    fmt.Sprintf("%d", maxAgents),
					"requests.memory": fmt.Sprintf("%dGi", maxAgents),
				},
			},
		},
	}
	_, _ = client.Resource(rqGVR).Namespace(namespace).Create(ctx, rq, metav1.CreateOptions{})

	npGVR := schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"}
	np := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.k8s.io/v1",
			"kind":       "NetworkPolicy",
			"metadata": map[string]interface{}{
				"name":      "tenant-isolation",
				"namespace": namespace,
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
										"arcana.io/tenant": tenantName,
									},
								},
							},
							map[string]interface{}{
								"namespaceSelector": map[string]interface{}{
									"matchLabels": map[string]interface{}{
										"kubernetes.io/metadata.name": "arcana",
									},
								},
							},
						},
					},
				},
				"egress": []interface{}{
					map[string]interface{}{},
				},
			},
		},
	}
	_, _ = client.Resource(npGVR).Namespace(namespace).Create(ctx, np, metav1.CreateOptions{})

	return nil
}

func registerTenantInMesh(name, displayName string, namespaces []string, maxAgents int) error {
	payload, _ := json.Marshal(map[string]interface{}{
		"id":           name,
		"name":         displayName,
		"namespace":    namespaces[0],
		"budget_limit": 0,
	})
	resp, err := httpClient.Post(
		fmt.Sprintf("http://%s:8083/api/v1/agents/register", meshHost),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func patchTenantPhase(ctx context.Context, client dynamic.Interface, name, phase, message string) {
	patch := []byte(fmt.Sprintf(
		`{"status":{"phase":"%s","message":"%s","lastUpdated":"%s"}}`,
		phase, message, time.Now().Format(time.RFC3339),
	))
	_, err := client.Resource(tenantGVR).Patch(
		ctx, name, "application/merge-patch+json", patch, metav1.PatchOptions{}, "status",
	)
	if err != nil {
		log.Printf("operator: patch tenant status %s/%s: %v", name, phase, err)
	}
}
