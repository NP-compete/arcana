package main

import (
	"context"
	"fmt"
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var backupGVR = schema.GroupVersionResource{
	Group:    "arcana.io",
	Version:  "v1alpha1",
	Resource: "arcanabackuppolicies",
}

func reconcileBackupPolicies(ctx context.Context, client dynamic.Interface) {
	policies, err := client.Resource(backupGVR).Namespace("arcana").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("operator: list backup policies failed: %v", err)
		return
	}

	for _, policy := range policies.Items {
		reconcileBackupPolicy(ctx, client, &policy)
	}

	if len(policies.Items) > 0 {
		log.Printf("operator: reconciled %d backup policies", len(policies.Items))
	}
}

func reconcileBackupPolicy(ctx context.Context, client dynamic.Interface, policy *unstructured.Unstructured) {
	name := policy.GetName()
	namespace := policy.GetNamespace()

	currentPhase, _, _ := unstructured.NestedString(policy.Object, "status", "phase")
	if currentPhase == "Active" {
		return
	}

	spec, _ := policy.Object["spec"].(map[string]interface{})
	if spec == nil {
		return
	}

	schedule, _ := spec["schedule"].(string)
	if schedule == "" {
		schedule = "0 */6 * * *"
	}

	targets, _ := toStringSlice(spec["targets"])
	if len(targets) == 0 {
		targets = []string{"postgresql", "skill-bank", "codex-indices"}
	}

	retention, _ := spec["retention"].(map[string]interface{})
	daily := 7
	if d, ok := retention["daily"].(float64); ok {
		daily = int(d)
	}

	storage, _ := spec["storage"].(map[string]interface{})
	primaryDest, _ := storage["primary"].(string)
	if primaryDest == "" {
		primaryDest = "s3://arcana-backups/default/"
	}

	log.Printf("operator: backup policy %s/%s configured (schedule=%s, targets=%v, retention=%dd, dest=%s)",
		namespace, name, schedule, targets, daily, primaryDest)

	patchBackupPhase(ctx, client, namespace, name, "Active",
		fmt.Sprintf("Schedule: %s, Targets: %v, Retention: %dd", schedule, targets, daily))
}

func patchBackupPhase(ctx context.Context, client dynamic.Interface, namespace, name, phase, message string) {
	patch := []byte(fmt.Sprintf(
		`{"status":{"phase":"%s","message":"%s","lastUpdated":"%s"}}`,
		phase, message, time.Now().Format(time.RFC3339),
	))
	_, err := client.Resource(backupGVR).Namespace(namespace).Patch(
		ctx, name, "application/merge-patch+json", patch, metav1.PatchOptions{}, "status",
	)
	if err != nil {
		log.Printf("operator: patch backup status %s/%s: %v", namespace, name, err)
	}
}
