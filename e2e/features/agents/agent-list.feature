@ui
Feature: Agent list and templates

  As a developer
  I want to see a list of my agents and available templates
  So that I can manage existing agents and quickly deploy from pre-built configurations

  Background:
    Given I am logged in as "admin"

  @smoke
  Scenario: Agents page loads and displays heading
    When I navigate to "/agents"
    Then I should see the "Agents" page

  Scenario: Agents page shows the Deploy button
    When I navigate to "/agents"
    Then I should see the "Deploy" button

  Scenario: Agents page displays agent templates
    When I navigate to "/agents"
    Then I should see "Templates"

  Scenario: Agent templates section shows pre-built options
    When I navigate to "/agents"
    Then I should see "Templates"
    And I should see the "Deploy" button

  @api
  Scenario: Agents API returns agent list
    Given I set the API role to "admin"
    When I send a GET request to "/api/v1/agents"
    Then the response should be successful
    And the response JSON should have property "agents"
