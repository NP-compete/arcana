package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// K8sExecutor spawns ephemeral Kubernetes pods for sandboxed code execution.
// Each execution request creates a new pod that runs the user code in an
// isolated container with restricted security context, resource limits, and
// no network access (enforced via NetworkPolicy).
type K8sExecutor struct {
	client    kubernetes.Interface
	namespace string
	runtime   string // "arcana-sandbox" or "arcana-sandbox-dev"
}

// NewK8sExecutor creates a K8sExecutor using in-cluster configuration.
// Returns an error if the process is not running inside a Kubernetes cluster.
func NewK8sExecutor() (*K8sExecutor, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("not running in cluster: %w", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create k8s client: %w", err)
	}

	ns := os.Getenv("SANDBOX_NAMESPACE")
	if ns == "" {
		ns = "arcana"
	}

	runtime := os.Getenv("SANDBOX_RUNTIME_CLASS")
	if runtime == "" {
		runtime = "arcana-sandbox-dev"
	}

	return &K8sExecutor{
		client:    client,
		namespace: ns,
		runtime:   runtime,
	}, nil
}

// Execute runs user code in an ephemeral Kubernetes pod and returns the
// captured stdout, stderr, and any execution error. The pod is always
// deleted after execution completes, regardless of success or failure.
func (e *K8sExecutor) Execute(ctx context.Context, language, code string, timeoutMs int) (string, string, error) {
	image, cmd := imageAndCmd(language, code)
	if image == "" {
		return "", "", fmt.Errorf("unsupported language: %s", language)
	}

	podName := fmt.Sprintf("sandbox-%d", time.Now().UnixNano())
	automount := false

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: e.namespace,
			Labels: map[string]string{
				"arcana.io/sandbox":  "true",
				"arcana.io/language": language,
			},
		},
		Spec: corev1.PodSpec{
			RuntimeClassName:             &e.runtime,
			RestartPolicy:                corev1.RestartPolicyNever,
			AutomountServiceAccountToken: &automount,
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot: boolPtr(true),
				RunAsUser:    int64Ptr(65534),
				RunAsGroup:   int64Ptr(65534),
				FSGroup:      int64Ptr(65534),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
			Containers: []corev1.Container{{
				Name:    "exec",
				Image:   image,
				Command: cmd,
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("64Mi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("50m"),
						corev1.ResourceMemory: resource.MustParse("32Mi"),
					},
				},
				SecurityContext: &corev1.SecurityContext{
					AllowPrivilegeEscalation: boolPtr(false),
					ReadOnlyRootFilesystem:   boolPtr(true),
					Capabilities: &corev1.Capabilities{
						Drop: []corev1.Capability{"ALL"},
					},
				},
			}},
			ActiveDeadlineSeconds: int64Ptr(int64(timeoutMs / 1000)),
		},
	}

	// Create pod
	created, err := e.client.CoreV1().Pods(e.namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return "", "", fmt.Errorf("create sandbox pod: %w", err)
	}

	// Ensure cleanup — always delete the pod regardless of outcome.
	defer func() {
		delCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = e.client.CoreV1().Pods(e.namespace).Delete(delCtx, created.Name, metav1.DeleteOptions{})
	}()

	// Wait for pod to complete
	if err := e.waitForPod(ctx, podName); err != nil {
		return "", "", err
	}

	// Get logs
	stdout, stderr := e.getPodLogs(ctx, podName)
	return stdout, stderr, nil
}

// waitForPod watches the pod until it reaches Succeeded or Failed phase,
// or until the context is cancelled.
func (e *K8sExecutor) waitForPod(ctx context.Context, name string) error {
	watcher, err := e.client.CoreV1().Pods(e.namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", name),
	})
	if err != nil {
		return fmt.Errorf("watch pod: %w", err)
	}
	defer watcher.Stop()

	for event := range watcher.ResultChan() {
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			continue
		}
		switch pod.Status.Phase {
		case corev1.PodSucceeded, corev1.PodFailed:
			return nil
		}
	}
	return fmt.Errorf("pod watch ended unexpectedly")
}

// getPodLogs retrieves the stdout from a completed pod's container logs.
// Output is capped at 1MB to prevent memory exhaustion.
func (e *K8sExecutor) getPodLogs(ctx context.Context, name string) (string, string) {
	req := e.client.CoreV1().Pods(e.namespace).GetLogs(name, &corev1.PodLogOptions{})
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Sprintf("failed to get logs: %v", err)
	}
	defer stream.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, io.LimitReader(stream, 1<<20)); err != nil {
		return buf.String(), fmt.Sprintf("error reading logs: %v", err)
	}
	return buf.String(), ""
}

// imageAndCmd maps a language name to the container image and command used
// to execute user code in the sandbox pod.
func imageAndCmd(language, code string) (string, []string) {
	switch language {
	case "python", "python3":
		return "python:3.12-alpine", []string{"python3", "-c", code}
	case "javascript", "js", "node":
		return "node:20-alpine", []string{"node", "-e", code}
	case "bash", "sh", "shell":
		return "alpine:3.19", []string{"sh", "-c", code}
	default:
		return "", nil
	}
}

func boolPtr(b bool) *bool    { return &b }
func int64Ptr(i int64) *int64 { return &i }
