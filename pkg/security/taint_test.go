package security

import (
	"crypto/ed25519"
	"testing"
)

func TestTaintTracker_Basic(t *testing.T) {
	tt := NewTaintTracker()

	if tt.IsTainted("data-1") {
		t.Error("expected not tainted initially")
	}

	tt.MarkTainted("data-1", TaintSecret)

	if !tt.IsTainted("data-1") {
		t.Error("expected tainted after marking")
	}

	labels := tt.GetLabels("data-1")
	if len(labels) != 1 || labels[0] != TaintSecret {
		t.Errorf("expected [secret], got %v", labels)
	}
}

func TestTaintTracker_Propagation(t *testing.T) {
	tt := NewTaintTracker()

	tt.MarkTainted("source", TaintPII)
	tt.MarkTainted("source", TaintCredential)
	tt.Propagate("source", "target")

	if !tt.IsTainted("target") {
		t.Error("expected target to be tainted after propagation")
	}

	labels := tt.GetLabels("target")
	if len(labels) != 2 {
		t.Errorf("expected 2 labels propagated, got %d", len(labels))
	}
}

func TestTaintTracker_SinkCheck(t *testing.T) {
	tt := NewTaintTracker()
	tt.MarkTainted("data", TaintSecret)
	tt.MarkTainted("data", TaintPII)

	if tt.CheckSink("data", []TaintLabel{TaintPII}) {
		t.Error("sink should fail — data has 'secret' label not in allowed list")
	}

	if !tt.CheckSink("data", []TaintLabel{TaintSecret, TaintPII}) {
		t.Error("sink should pass — all labels are allowed")
	}

	if !tt.CheckSink("clean-data", nil) {
		t.Error("untainted data should always pass sink check")
	}
}

func TestTaintTracker_Zeroize(t *testing.T) {
	tt := NewTaintTracker()
	tt.MarkTainted("secret-key", TaintSecret)

	if !tt.IsTainted("secret-key") {
		t.Error("expected tainted")
	}

	tt.Zeroize("secret-key")

	if tt.IsTainted("secret-key") {
		t.Error("expected not tainted after zeroization")
	}
}

func TestTaintTracker_Stats(t *testing.T) {
	tt := NewTaintTracker()
	tt.MarkTainted("a", TaintSecret)
	tt.MarkTainted("b", TaintPII)
	tt.MarkTainted("c", TaintSecret)

	stats := tt.Stats()
	if stats["tracked_items"].(int) != 3 {
		t.Errorf("expected 3 tracked items, got %v", stats["tracked_items"])
	}
}

func TestSignedManifest(t *testing.T) {
	pub, priv, err := generateTestKeyPair()
	if err != nil {
		t.Fatalf("key generation: %v", err)
	}

	manifest := map[string]interface{}{
		"agent_name":   "test-agent",
		"version":      "1.0.0",
		"capabilities": []interface{}{"search", "code"},
		"protocols":    []interface{}{"mcp", "a2a"},
	}

	signed := SignManifest(manifest, priv, "test-signer")

	if signed.AgentName != "test-agent" {
		t.Errorf("expected test-agent, got %s", signed.AgentName)
	}
	if signed.Checksum == "" {
		t.Error("expected non-empty checksum")
	}
	if signed.Signature == "" {
		t.Error("expected non-empty signature")
	}
	if signed.SignedBy != "test-signer" {
		t.Errorf("expected test-signer, got %s", signed.SignedBy)
	}

	if !VerifyManifest(signed, pub) {
		t.Error("expected valid signature")
	}
}

func generateTestKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(nil)
	return pub, priv, err
}
