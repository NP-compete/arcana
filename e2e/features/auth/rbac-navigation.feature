@ui
Feature: Role-based navigation visibility

  As a platform operator
  I want each persona to see only the navigation items they are authorized for
  So that users cannot discover features outside their access scope

  # ── Admin: full access to all 18 items ──

  @smoke
  Scenario: Admin sees all navigation items
    Given I am logged in as "admin"
    Then the navigation should show "Dashboard"
    And the navigation should show "Chat"
    And the navigation should show "Build Hub"
    And the navigation should show "Org Chart"
    And the navigation should show "Agents"
    And the navigation should show "Connectors"
    And the navigation should show "MCP"
    And the navigation should show "Models"
    And the navigation should show "Skills"
    And the navigation should show "Flow Builder"
    And the navigation should show "Editor"
    And the navigation should show "Marketplace"
    And the navigation should show "Evaluations"
    And the navigation should show "Guardrails"
    And the navigation should show "FinOps"
    And the navigation should show "Audit"
    And the navigation should show "Approvals"
    And the navigation should show "Settings"

  # ── User: minimal access ──

  @smoke
  Scenario: User sees only consumer-facing items
    Given I am logged in as "user"
    Then the navigation should show "Dashboard"
    And the navigation should show "Chat"
    And the navigation should show "Agents"
    And the navigation should show "Marketplace"

  Scenario: User does not see builder or admin items
    Given I am logged in as "user"
    Then the navigation should not show "Build Hub"
    And the navigation should not show "Org Chart"
    And the navigation should not show "Connectors"
    And the navigation should not show "MCP"
    And the navigation should not show "Models"
    And the navigation should not show "Skills"
    And the navigation should not show "Flow Builder"
    And the navigation should not show "Editor"
    And the navigation should not show "Evaluations"
    And the navigation should not show "Guardrails"
    And the navigation should not show "FinOps"
    And the navigation should not show "Audit"
    And the navigation should not show "Approvals"
    And the navigation should not show "Settings"

  # ── Developer: builder tools, no ops/compliance ──

  Scenario: Developer sees builder tools
    Given I am logged in as "developer"
    Then the navigation should show "Build Hub"
    And the navigation should show "Connectors"
    And the navigation should show "MCP"
    And the navigation should show "Models"
    And the navigation should show "Skills"
    And the navigation should show "Flow Builder"
    And the navigation should show "Editor"
    And the navigation should show "Evaluations"
    And the navigation should show "Guardrails"

  Scenario: Developer does not see ops or compliance items
    Given I am logged in as "developer"
    Then the navigation should not show "Org Chart"
    And the navigation should not show "FinOps"
    And the navigation should not show "Audit"
    And the navigation should not show "Approvals"
    And the navigation should not show "Settings"

  # ── Data Engineer: data-focused subset ──

  Scenario: Data Engineer sees data pipeline tools
    Given I am logged in as "data-engineer"
    Then the navigation should show "Connectors"
    And the navigation should show "MCP"
    And the navigation should show "Models"

  Scenario: Data Engineer does not see builder or admin items
    Given I am logged in as "data-engineer"
    Then the navigation should not show "Build Hub"
    And the navigation should not show "Skills"
    And the navigation should not show "Flow Builder"
    And the navigation should not show "Editor"
    And the navigation should not show "Evaluations"
    And the navigation should not show "Guardrails"
    And the navigation should not show "FinOps"
    And the navigation should not show "Audit"
    And the navigation should not show "Settings"

  # ── SRE: operations and cost management ──

  Scenario: SRE sees operations items
    Given I am logged in as "sre"
    Then the navigation should show "Org Chart"
    And the navigation should show "FinOps"
    And the navigation should show "Settings"

  Scenario: SRE does not see builder or compliance items
    Given I am logged in as "sre"
    Then the navigation should not show "Build Hub"
    And the navigation should not show "Connectors"
    And the navigation should not show "MCP"
    And the navigation should not show "Models"
    And the navigation should not show "Skills"
    And the navigation should not show "Flow Builder"
    And the navigation should not show "Editor"
    And the navigation should not show "Evaluations"
    And the navigation should not show "Guardrails"
    And the navigation should not show "Audit"

  # ── Auditor: compliance and read-only ──

  Scenario: Auditor sees compliance items
    Given I am logged in as "auditor"
    Then the navigation should show "Audit"
    And the navigation should show "Settings"

  Scenario: Auditor does not see builder or ops items
    Given I am logged in as "auditor"
    Then the navigation should not show "Build Hub"
    And the navigation should not show "Org Chart"
    And the navigation should not show "Connectors"
    And the navigation should not show "MCP"
    And the navigation should not show "Models"
    And the navigation should not show "Skills"
    And the navigation should not show "Flow Builder"
    And the navigation should not show "Editor"
    And the navigation should not show "Evaluations"
    And the navigation should not show "Guardrails"
    And the navigation should not show "FinOps"
    And the navigation should not show "Approvals"
