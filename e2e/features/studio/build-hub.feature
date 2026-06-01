@ui
Feature: Build hub creation center

  As a developer
  I want a central hub for creating platform resources
  So that I can quickly scaffold new agents, skills, models, and tools

  Background:
    Given I am logged in as "admin"
    When I navigate to "/build-hub"

  @smoke
  Scenario: Build hub page loads with heading
    Then I should see the "Build Hub" page

  Scenario: Build hub shows Agent creation card
    Then I should see "Agent"

  Scenario: Build hub shows Skill creation card
    Then I should see "Skill"

  Scenario: Build hub shows Model creation card
    Then I should see "Model"

  Scenario: Build hub shows all five creation options
    Then I should see "Agent"
    And I should see "Skill"
    And I should see "Model"
    And I should see "MCP"
    And I should see "Subagent"
