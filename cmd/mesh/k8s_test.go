package main

import (
	"testing"
)

func TestExtractPodHealth_NoPods(t *testing.T) {
	info := ExtractPodHealth(nil)
	if info.Phase != "NoPods" {
		t.Errorf("expected NoPods, got %s", info.Phase)
	}
	if info.Ready {
		t.Error("expected not ready for no pods")
	}
}

func TestExtractPodHealth_RunningPod(t *testing.T) {
	pods := []map[string]interface{}{
		{
			"status": map[string]interface{}{
				"phase": "Running",
				"containerStatuses": []interface{}{
					map[string]interface{}{
						"name":         "agent",
						"restartCount": float64(0),
						"ready":        true,
						"state":        map[string]interface{}{"running": map[string]interface{}{}},
					},
				},
			},
		},
	}

	info := ExtractPodHealth(pods)
	if info.Phase != "Running" {
		t.Errorf("expected Running, got %s", info.Phase)
	}
	if !info.Ready {
		t.Error("expected ready")
	}
	if info.RestartCount != 0 {
		t.Errorf("expected 0 restarts, got %d", info.RestartCount)
	}
	if info.FailureReason != "" {
		t.Errorf("expected no failure reason, got %s", info.FailureReason)
	}
}

func TestExtractPodHealth_CrashLoopBackOff(t *testing.T) {
	pods := []map[string]interface{}{
		{
			"status": map[string]interface{}{
				"phase": "Running",
				"containerStatuses": []interface{}{
					map[string]interface{}{
						"name":         "agent",
						"restartCount": float64(5),
						"ready":        false,
						"state": map[string]interface{}{
							"waiting": map[string]interface{}{
								"reason":  "CrashLoopBackOff",
								"message": "back-off 5m0s restarting failed container",
							},
						},
					},
				},
			},
		},
	}

	info := ExtractPodHealth(pods)
	if info.FailureReason != "CrashLoopBackOff" {
		t.Errorf("expected CrashLoopBackOff, got %s", info.FailureReason)
	}
	if info.RestartCount != 5 {
		t.Errorf("expected 5 restarts, got %d", info.RestartCount)
	}
	if info.Ready {
		t.Error("expected not ready")
	}
}

func TestExtractPodHealth_MultiPod(t *testing.T) {
	pods := []map[string]interface{}{
		{
			"status": map[string]interface{}{
				"phase": "Running",
				"containerStatuses": []interface{}{
					map[string]interface{}{"restartCount": float64(2), "ready": true, "state": map[string]interface{}{}},
				},
			},
		},
		{
			"status": map[string]interface{}{
				"phase": "Pending",
				"containerStatuses": []interface{}{
					map[string]interface{}{"restartCount": float64(1), "ready": false, "state": map[string]interface{}{}},
				},
			},
		},
	}

	info := ExtractPodHealth(pods)
	if info.Phase != "Pending" {
		t.Errorf("expected worst-case Pending, got %s", info.Phase)
	}
	if info.Ready {
		t.Error("expected not ready when any pod is not ready")
	}
	if info.RestartCount != 3 {
		t.Errorf("expected 3 total restarts, got %d", info.RestartCount)
	}
}
