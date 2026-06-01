@api @smoke
Feature: Guardrail rule management

  As a platform administrator
  I want to create and list guardrail rules per agent
  So that each agent has tailored content moderation policies

  Background:
    Given an agent named "guard-rules-bdd" exists

  # ── Create rules ──

  Scenario: Create a regex-based guardrail rule
    When I create a guardrail rule of type "regex" with pattern "sk-[a-zA-Z0-9]{32}" for agent "guard-rules-bdd"
    Then the response should be successful
    And the rule should be created with an ID

  Scenario: Create a keyword-based guardrail rule
    When I create a guardrail rule of type "keyword" with pattern "CONFIDENTIAL" for agent "guard-rules-bdd"
    Then the response should be successful
    And the rule should be created with an ID

  # ── List rules ──

  Scenario: List rules for an agent includes previously created rules
    When I create a guardrail rule of type "regex" with pattern "password:\\s*.+" for agent "guard-rules-bdd"
    And I list guardrail rules for agent "guard-rules-bdd"
    Then the response should be successful
    And the rules list should contain agent "guard-rules-bdd"

  # ── Rule scoping ──

  Scenario: Rules are scoped to their agent
    Given an agent named "guard-other-bdd" exists
    When I create a guardrail rule of type "keyword" with pattern "SECRET_TOKEN" for agent "guard-rules-bdd"
    And I list guardrail rules for agent "guard-rules-bdd"
    Then the rules list should contain agent "guard-rules-bdd"
