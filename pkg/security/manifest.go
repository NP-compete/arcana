package security

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

type SignedManifest struct {
	AgentName    string                 `json:"agent_name"`
	Version      string                 `json:"version"`
	Capabilities []string               `json:"capabilities"`
	Protocols    []string               `json:"protocols"`
	Checksum     string                 `json:"checksum"`
	Signature    string                 `json:"signature"`
	SignedAt     time.Time              `json:"signed_at"`
	SignedBy     string                 `json:"signed_by"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

func SignManifest(manifest map[string]interface{}, privateKey ed25519.PrivateKey, signer string) SignedManifest {
	payload, _ := json.Marshal(manifest)
	checksum := sha256.Sum256(payload)
	sig := ed25519.Sign(privateKey, payload)

	return SignedManifest{
		AgentName:    getString(manifest, "agent_name"),
		Version:      getString(manifest, "version"),
		Capabilities: getStringSlice(manifest, "capabilities"),
		Protocols:    getStringSlice(manifest, "protocols"),
		Checksum:     hex.EncodeToString(checksum[:]),
		Signature:    hex.EncodeToString(sig),
		SignedAt:     time.Now().UTC(),
		SignedBy:     signer,
		Metadata:     manifest,
	}
}

func VerifyManifest(manifest SignedManifest, publicKey ed25519.PublicKey) bool {
	payload, _ := json.Marshal(manifest.Metadata)
	sig, err := hex.DecodeString(manifest.Signature)
	if err != nil {
		return false
	}
	return ed25519.Verify(publicKey, payload, sig)
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getStringSlice(m map[string]interface{}, key string) []string {
	arr, ok := m[key].([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
