@ui
Feature: Flow builder canvas

  As a developer
  I want to design agent workflows visually
  So that I can compose multi-step pipelines without writing code

  Background:
    Given I am logged in as "admin"
    When I navigate to "/flow-builder"

  @smoke
  Scenario: Flow builder page loads with heading
    Then I should see the "Flow Builder" page

  Scenario: Canvas area is visible
    Then I should see a chart or visualization

  Scenario: Node palette is visible
    Then I should see "Nodes"

  Scenario: Save and export buttons are available
    Then I should see the "Save" button
    And I should see the "Export" button

  Scenario: Flow builder matches visual baseline
    Then the page should match the visual baseline "studio-flow-builder"
