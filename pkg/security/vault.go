package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type VaultClient struct {
	addr   string
	token  string
	client *http.Client
}

func NewVaultClient() *VaultClient {
	addr := os.Getenv("VAULT_ADDR")
	if addr == "" {
		addr = "http://vault.arcana.svc.cluster.local:8200"
	}
	return &VaultClient{
		addr:   addr,
		token:  os.Getenv("VAULT_TOKEN"),
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (vc *VaultClient) Available() bool {
	return vc.token != ""
}

func (vc *VaultClient) CreateTenantKey(tenant string) (string, error) {
	if !vc.Available() {
		return "", fmt.Errorf("vault not configured")
	}

	path := fmt.Sprintf("/v1/transit/keys/arcana-tenant-%s", tenant)
	payload, _ := json.Marshal(map[string]interface{}{
		"type":                "aes256-gcm96",
		"deletion_allowed":   false,
		"exportable":         false,
		"allow_plaintext_backup": false,
	})

	resp, err := vc.doRequest("POST", path, payload)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("vault: create key failed: %s", string(body))
	}

	keyName := fmt.Sprintf("arcana-tenant-%s", tenant)
	log.Printf("vault: created encryption key for tenant %s", tenant)
	return keyName, nil
}

func (vc *VaultClient) Encrypt(tenant string, plaintext []byte) (string, error) {
	if !vc.Available() {
		return "", fmt.Errorf("vault not configured")
	}

	path := fmt.Sprintf("/v1/transit/encrypt/arcana-tenant-%s", tenant)
	payload, _ := json.Marshal(map[string]interface{}{
		"plaintext": plaintext,
	})

	resp, err := vc.doRequest("POST", path, payload)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Ciphertext string `json:"ciphertext"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Data.Ciphertext, nil
}

func (vc *VaultClient) Decrypt(tenant, ciphertext string) ([]byte, error) {
	if !vc.Available() {
		return nil, fmt.Errorf("vault not configured")
	}

	path := fmt.Sprintf("/v1/transit/decrypt/arcana-tenant-%s", tenant)
	payload, _ := json.Marshal(map[string]interface{}{
		"ciphertext": ciphertext,
	})

	resp, err := vc.doRequest("POST", path, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Plaintext []byte `json:"plaintext"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Data.Plaintext, nil
}

func (vc *VaultClient) RotateTenantKey(tenant string) error {
	if !vc.Available() {
		return fmt.Errorf("vault not configured")
	}

	path := fmt.Sprintf("/v1/transit/keys/arcana-tenant-%s/rotate", tenant)
	resp, err := vc.doRequest("POST", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	log.Printf("vault: rotated encryption key for tenant %s", tenant)
	return nil
}

func (vc *VaultClient) StoreTenantSecret(tenant, key, value string) error {
	if !vc.Available() {
		return fmt.Errorf("vault not configured")
	}

	path := fmt.Sprintf("/v1/secret/data/arcana/tenants/%s/%s", tenant, key)
	payload, _ := json.Marshal(map[string]interface{}{
		"data": map[string]string{"value": value},
	})

	resp, err := vc.doRequest("POST", path, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (vc *VaultClient) GetTenantSecret(tenant, key string) (string, error) {
	if !vc.Available() {
		return "", fmt.Errorf("vault not configured")
	}

	path := fmt.Sprintf("/v1/secret/data/arcana/tenants/%s/%s", tenant, key)
	resp, err := vc.doRequest("GET", path, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Data map[string]string `json:"data"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Data.Data["value"], nil
}

func (vc *VaultClient) doRequest(method, path string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, vc.addr+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Vault-Token", vc.token)
	req.Header.Set("Content-Type", "application/json")
	return vc.client.Do(req)
}
