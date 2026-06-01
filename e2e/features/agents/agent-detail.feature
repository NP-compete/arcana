@ui
Feature: Agent detail page

  As a developer
  I want to view an agent's detail page
  So that I can inspect its configuration, memory, and guardrails

  Background:
    Given I am logged in as "admin"
    And an agent named "bdd-detail-agent" exists

  @smoke
  Scenario: Agent detail page loads and displays agent name
    When I open the agent detail page for "bdd-detail-agent"
    Then the agent detail should show "bdd-detail-agent"

  Scenario: Agent detail page shows Open Chat button
    When I open the agent detail page for "bdd-detail-agent"
    Then I should see the "Open Chat" button

  Scenario: Agent detail page shows Memory section
    When I open the agent detail page for "bdd-detail-agent"
    Then the agent detail should show "Memory"

  Scenario: Agent detail page shows Guardrails section
    When I open the agent detail page for "bdd-detail-agent"
    Then the agent detail should show "Guardrails"
