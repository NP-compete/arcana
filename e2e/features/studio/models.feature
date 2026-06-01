@ui
Feature: Model registry

  As a developer
  I want to view and register AI models
  So that I can manage the models available for agent deployment

  Background:
    Given I am logged in as "admin"
    When I navigate to "/models"

  @smoke
  Scenario: Models page loads with heading
    Then I should see the "Models" page

  Scenario: Register model button is visible
    Then I should see the "Register Model" button

  Scenario: Models page displays model table
    Then I should see a table with at least 1 rows

  Scenario: Model table shows provider information
    Then the table should contain "Provider"

  Scenario: Models page matches visual baseline
    Then the page should match the visual baseline "studio-models"
