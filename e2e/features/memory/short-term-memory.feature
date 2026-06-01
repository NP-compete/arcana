@api @smoke
Feature: Short-term memory storage and retrieval

  As a platform agent
  I want to store and retrieve key-value pairs with TTL
  So that I can maintain ephemeral context during a conversation

  Background:
    Given an agent named "mem-short-bdd" exists

  # ── Store and retrieve ──

  Scenario: Store a single short-term memory entry
    When I store short-term memory for agent "mem-short-bdd" with key "user_lang" and value "en-US"
    Then the response should be successful

  Scenario: Retrieve short-term memory returns stored entries
    When I store short-term memory for agent "mem-short-bdd" with key "topic" and value "kubernetes"
    And I retrieve short-term memory for agent "mem-short-bdd"
    Then the response should be successful
    And the short-term memory for agent "mem-short-bdd" should have 1 entries

  # ── Multiple entries ──

  Scenario: Store multiple entries and verify count
    When I store short-term memory for agent "mem-short-bdd" with key "ctx_model" and value "gpt-4"
    And I store short-term memory for agent "mem-short-bdd" with key "ctx_temp" and value "0.7"
    And I store short-term memory for agent "mem-short-bdd" with key "ctx_tokens" and value "4096"
    Then the short-term memory for agent "mem-short-bdd" should have 3 entries

  # ── Isolation ──

  Scenario: Short-term memory is scoped to the agent
    When I store short-term memory for agent "mem-short-bdd" with key "scoped_key" and value "scoped_val"
    And I retrieve short-term memory for agent "mem-short-bdd"
    Then the response should be successful
    And the short-term memory for agent "mem-short-bdd" should have 1 entries
