package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type SCIMUser struct {
	ID          string `json:"id"`
	UserName    string `json:"userName"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Active      bool   `json:"active"`
	Roles       []string `json:"roles"`
	CreatedAt   string `json:"meta.created,omitempty"`
}

type SCIMStore struct {
	mu    sync.RWMutex
	users map[string]*SCIMUser
}

var scimStore = &SCIMStore{users: make(map[string]*SCIMUser)}

func handleSCIMUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")

	switch r.Method {
	case http.MethodGet:
		scimStore.mu.RLock()
		users := make([]SCIMUser, 0, len(scimStore.users))
		for _, u := range scimStore.users {
			users = append(users, *u)
		}
		scimStore.mu.RUnlock()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"schemas":      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
			"totalResults": len(users),
			"Resources":    users,
		})

	case http.MethodPost:
		var user SCIMUser
		json.NewDecoder(r.Body).Decode(&user)
		if user.UserName == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"detail": "userName required"})
			return
		}
		user.ID = fmt.Sprintf("scim-%d", time.Now().UnixNano())
		user.Active = true
		user.CreatedAt = time.Now().UTC().Format(time.RFC3339)
		if len(user.Roles) == 0 {
			user.Roles = []string{"user"}
		}
		scimStore.mu.Lock()
		scimStore.users[user.ID] = &user
		scimStore.mu.Unlock()
		log.Printf("scim: provisioned user %s (%s)", user.UserName, user.Email)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleSCIMUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")

	id := r.URL.Path[len("/scim/v2/Users/"):]
	scimStore.mu.RLock()
	user, ok := scimStore.users[id]
	scimStore.mu.RUnlock()

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"detail": "User not found"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(user)

	case http.MethodPut:
		var updated SCIMUser
		json.NewDecoder(r.Body).Decode(&updated)
		updated.ID = id
		scimStore.mu.Lock()
		scimStore.users[id] = &updated
		scimStore.mu.Unlock()
		log.Printf("scim: updated user %s", updated.UserName)
		json.NewEncoder(w).Encode(updated)

	case http.MethodDelete:
		scimStore.mu.Lock()
		delete(scimStore.users, id)
		scimStore.mu.Unlock()
		log.Printf("scim: deprovisioned user %s", id)
		w.WriteHeader(http.StatusNoContent)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
