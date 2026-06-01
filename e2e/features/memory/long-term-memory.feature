@api @smoke
Feature: Long-term memory with embeddings and semantic search

  As a platform agent
  I want to store long-term memories with vector embeddings
  So that I can perform semantic retrieval of past knowledge

  Background:
    Given an agent named "mem-long-bdd" exists

  # ── Store with embedding ──

  Scenario: Store long-term memory and receive a 64-dimension embedding
    When I store long-term memory for agent "mem-long-bdd" with content "The user prefers dark mode and compact layout"
    Then the response should be successful
    And the long-term memory embedding should have 64 dimensions

  Scenario: Store multiple long-term memories
    When I store long-term memory for agent "mem-long-bdd" with content "User works at Acme Corp in the platform engineering team"
    And I store long-term memory for agent "mem-long-bdd" with content "User prefers Helm charts over raw Kubernetes manifests"
    Then the response should be successful

  # ── Semantic search ──

  Scenario: Search returns relevant results with similarity scores
    When I store long-term memory for agent "mem-long-bdd" with content "The deployment pipeline uses ArgoCD for GitOps"
    And I store long-term memory for agent "mem-long-bdd" with content "Monitoring stack is Prometheus plus Grafana"
    And I search long-term memory for agent "mem-long-bdd" with query "GitOps deployment"
    Then the response should be successful
    And the search results should have at least 1 entries
    And the search results should have scores

  Scenario: Search with no matching content returns results gracefully
    When I store long-term memory for agent "mem-long-bdd" with content "The database engine is PostgreSQL 16"
    And I search long-term memory for agent "mem-long-bdd" with query "completely unrelated quantum physics"
    Then the response should be successful
