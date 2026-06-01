@ui
Feature: Evaluation framework

  As a developer
  I want to run and review agent evaluations
  So that I can measure and improve agent quality over time

  Background:
    Given I am logged in as "admin"
    When I navigate to "/evaluations"

  @smoke
  Scenario: Evaluations page loads with heading
    Then I should see the "Evaluations" page

  Scenario: Eval runs tab is visible
    Then the "Runs" tab should be active

  Scenario: Quality dashboard tab loads
    When I open the "Quality" tab
    Then the "Quality" tab should be active
    And I should see a chart or visualization

  Scenario: Eval builder tab shows wizard steps
    When I open the "Builder" tab
    Then the "Builder" tab should be active
    And I should see "Create Evaluation"

  Scenario: Evaluations page matches visual baseline
    Then the page should match the visual baseline "studio-evaluations"
