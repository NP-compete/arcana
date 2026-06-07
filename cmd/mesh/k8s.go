package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type K8sClient struct {
	host   string
	token  string
	caCert string
	client *http.Client
}

func NewK8sClient() (*K8sClient, error) {
	host := os.Getenv("KUBERNETES_SERVICE_HOST")
	port := os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		return nil, fmt.Errorf("not running in cluster")
	}

	tokenBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return nil, fmt.Errorf("cannot read SA token: %w", err)
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	caCert, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err == nil {
		_ = caCert
	}

	return &K8sClient{
		host:  fmt.Sprintf("https://%s:%s", host, port),
		token: strings.TrimSpace(string(tokenBytes)),
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: &http.Transport{TLSClientConfig: tlsConfig},
		},
	}, nil
}

func (k *K8sClient) do(method, path string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, k.host+path, reqBody)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+k.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return respBody, resp.StatusCode, nil
}

func AgentNamespace(agentName string) string {
	return "arcana-agent-" + agentName
}

func (k *K8sClient) CreateNamespace(name string, labels map[string]string) error {
	ns := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Namespace",
		"metadata": map[string]interface{}{
			"name":   name,
			"labels": labels,
		},
	}
	_, code, err := k.do("POST", "/api/v1/namespaces", ns)
	if err != nil {
		return err
	}
	if code == 409 {
		return nil
	}
	if code >= 300 {
		return fmt.Errorf("create namespace %s: HTTP %d", name, code)
	}
	return nil
}

func (k *K8sClient) CreateConfigMap(namespace, name string, data map[string]string) error {
	cm := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
		},
		"data": data,
	}
	_, code, err := k.do("POST", fmt.Sprintf("/api/v1/namespaces/%s/configmaps", namespace), cm)
	if err != nil {
		return err
	}
	if code == 409 {
		return nil
	}
	if code >= 300 {
		return fmt.Errorf("create configmap: HTTP %d", code)
	}
	return nil
}

func (k *K8sClient) CreateResourceQuota(namespace, name string, hard map[string]string) error {
	rq := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ResourceQuota",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
		},
		"spec": map[string]interface{}{
			"hard": hard,
		},
	}
	_, code, err := k.do("POST", fmt.Sprintf("/api/v1/namespaces/%s/resourcequotas", namespace), rq)
	if err != nil {
		return err
	}
	if code == 409 {
		return nil
	}
	if code >= 300 {
		return fmt.Errorf("create resourcequota: HTTP %d", code)
	}
	return nil
}

func (k *K8sClient) CreateNetworkPolicy(namespace, name string) error {
	np := map[string]interface{}{
		"apiVersion": "networking.k8s.io/v1",
		"kind":       "NetworkPolicy",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
		},
		"spec": map[string]interface{}{
			"podSelector": map[string]interface{}{},
			"policyTypes": []string{"Ingress", "Egress"},
			"ingress": []map[string]interface{}{
				{
					"from": []map[string]interface{}{
						{"namespaceSelector": map[string]interface{}{
							"matchLabels": map[string]string{
								"app.kubernetes.io/part-of": "arcana",
							},
						}},
					},
				},
			},
			"egress": []map[string]interface{}{
				{
					"to": []map[string]interface{}{
						{"namespaceSelector": map[string]interface{}{
							"matchLabels": map[string]string{
								"app.kubernetes.io/part-of": "arcana",
							},
						}},
					},
				},
				{
					"to": []map[string]interface{}{
						{"namespaceSelector": map[string]interface{}{}},
					},
					"ports": []map[string]interface{}{
						{"protocol": "TCP", "port": 53},
						{"protocol": "UDP", "port": 53},
					},
				},
			},
		},
	}
	_, code, err := k.do("POST", fmt.Sprintf("/apis/networking.k8s.io/v1/namespaces/%s/networkpolicies", namespace), np)
	if err != nil {
		return err
	}
	if code == 409 {
		return nil
	}
	if code >= 300 {
		return fmt.Errorf("create networkpolicy: HTTP %d", code)
	}
	return nil
}

func (k *K8sClient) CreateArcanaAgent(namespace, name string, spec map[string]interface{}) error {
	agent := map[string]interface{}{
		"apiVersion": "arcana.io/v1alpha1",
		"kind":       "ArcanaAgent",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
		},
		"spec": spec,
	}
	_, code, err := k.do("POST",
		fmt.Sprintf("/apis/arcana.io/v1alpha1/namespaces/%s/arcanaagents", namespace), agent)
	if err != nil {
		return err
	}
	if code == 409 {
		return nil
	}
	if code >= 300 {
		return fmt.Errorf("create arcanaagent: HTTP %d", code)
	}
	return nil
}

func (k *K8sClient) CreateSecret(namespace, name string, data map[string]string) error {
	secret := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
		},
		"type":       "Opaque",
		"stringData": data,
	}
	_, code, err := k.do("POST",
		fmt.Sprintf("/api/v1/namespaces/%s/secrets", namespace), secret)
	if err != nil {
		return err
	}
	if code == 409 {
		return nil
	}
	if code >= 300 {
		return fmt.Errorf("create secret: HTTP %d", code)
	}
	return nil
}

func (k *K8sClient) CreateDeployment(namespace, name string, spec map[string]interface{}) error {
	deployment := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"labels": map[string]string{
				"app":                          name,
				"app.kubernetes.io/part-of":    "arcana",
				"app.kubernetes.io/managed-by": "arcana-mesh",
			},
		},
		"spec": spec,
	}
	_, code, err := k.do("POST",
		fmt.Sprintf("/apis/apps/v1/namespaces/%s/deployments", namespace), deployment)
	if err != nil {
		return err
	}
	if code == 409 {
		return nil
	}
	if code >= 300 {
		return fmt.Errorf("create deployment: HTTP %d", code)
	}
	return nil
}

func (k *K8sClient) CreateService(namespace, name string, port int, targetPort int) error {
	svc := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
		},
		"spec": map[string]interface{}{
			"selector": map[string]string{
				"app": name,
			},
			"ports": []map[string]interface{}{
				{
					"port":       port,
					"targetPort": targetPort,
					"protocol":   "TCP",
					"name":       "http",
				},
			},
		},
	}
	_, code, err := k.do("POST",
		fmt.Sprintf("/api/v1/namespaces/%s/services", namespace), svc)
	if err != nil {
		return err
	}
	if code == 409 {
		return nil
	}
	if code >= 300 {
		return fmt.Errorf("create service: HTTP %d", code)
	}
	return nil
}

func (k *K8sClient) GetDeploymentStatus(namespace, name string) (map[string]interface{}, error) {
	body, code, err := k.do("GET",
		fmt.Sprintf("/apis/apps/v1/namespaces/%s/deployments/%s", namespace, name), nil)
	if err != nil {
		return nil, err
	}
	if code == 404 {
		return nil, nil
	}
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	return result, nil
}

func (k *K8sClient) GetNamespace(name string) (map[string]interface{}, error) {
	body, code, err := k.do("GET", "/api/v1/namespaces/"+name, nil)
	if err != nil {
		return nil, err
	}
	if code == 404 {
		return nil, nil
	}
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	return result, nil
}

func (k *K8sClient) ListNamespaceResources(namespace string) (map[string]int, error) {
	counts := map[string]int{}

	for _, resource := range []string{"configmaps", "resourcequotas"} {
		body, code, err := k.do("GET", fmt.Sprintf("/api/v1/namespaces/%s/%s", namespace, resource), nil)
		if err != nil || code != 200 {
			continue
		}
		var list map[string]interface{}
		json.Unmarshal(body, &list)
		if items, ok := list["items"].([]interface{}); ok {
			counts[resource] = len(items)
		}
	}

	body, code, err := k.do("GET",
		fmt.Sprintf("/apis/networking.k8s.io/v1/namespaces/%s/networkpolicies", namespace), nil)
	if err == nil && code == 200 {
		var list map[string]interface{}
		json.Unmarshal(body, &list)
		if items, ok := list["items"].([]interface{}); ok {
			counts["networkpolicies"] = len(items)
		}
	}

	body, code, err = k.do("GET",
		fmt.Sprintf("/apis/arcana.io/v1alpha1/namespaces/%s/arcanaagents", namespace), nil)
	if err == nil && code == 200 {
		var list map[string]interface{}
		json.Unmarshal(body, &list)
		if items, ok := list["items"].([]interface{}); ok {
			counts["arcanaagents"] = len(items)
		}
	}

	return counts, nil
}

func (k *K8sClient) ListPods(namespace, labelSelector string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods?labelSelector=%s", namespace, strings.ReplaceAll(labelSelector, " ", "%20"))
	body, code, err := k.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, fmt.Errorf("list pods: HTTP %d", code)
	}
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	items, ok := result["items"].([]interface{})
	if !ok {
		return nil, nil
	}
	pods := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if pod, ok := item.(map[string]interface{}); ok {
			pods = append(pods, pod)
		}
	}
	return pods, nil
}

func (k *K8sClient) ScaleDeployment(namespace, name string, replicas int) error {
	scale := map[string]interface{}{
		"spec": map[string]interface{}{
			"replicas": replicas,
		},
	}
	patch, _ := json.Marshal(scale)
	req, err := http.NewRequest("PATCH",
		k.host+fmt.Sprintf("/apis/apps/v1/namespaces/%s/deployments/%s", namespace, name),
		bytes.NewReader(patch))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+k.token)
	req.Header.Set("Content-Type", "application/strategic-merge-patch+json")
	resp, err := k.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("scale deployment %s/%s to %d: HTTP %d", namespace, name, replicas, resp.StatusCode)
	}
	return nil
}

func ExtractPodHealth(pods []map[string]interface{}) PodHealthInfo {
	info := PodHealthInfo{Phase: "Unknown"}
	if len(pods) == 0 {
		info.Phase = "NoPods"
		return info
	}
	// Assume ready until proven otherwise
	info.Ready = true
	allRunning := true
	for _, pod := range pods {
		status, _ := pod["status"].(map[string]interface{})
		if status == nil {
			info.Ready = false
			allRunning = false
			continue
		}
		if phase, ok := status["phase"].(string); ok {
			if phase != "Running" {
				allRunning = false
				info.Phase = phase
			}
		}
		podReady := false
		containers, _ := status["containerStatuses"].([]interface{})
		for _, c := range containers {
			cs, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			if rc, ok := cs["restartCount"].(float64); ok {
				info.RestartCount += int(rc)
			}
			if ready, ok := cs["ready"].(bool); ok && ready {
				podReady = true
			}
			if state, ok := cs["state"].(map[string]interface{}); ok {
				if waiting, ok := state["waiting"].(map[string]interface{}); ok {
					if reason, ok := waiting["reason"].(string); ok {
						info.FailureReason = reason
					}
				}
				if terminated, ok := state["terminated"].(map[string]interface{}); ok {
					if reason, ok := terminated["reason"].(string); ok && info.FailureReason == "" {
						info.FailureReason = reason
					}
				}
			}
		}
		if !podReady {
			info.Ready = false
		}
	}
	if allRunning {
		info.Phase = "Running"
	}
	return info
}
