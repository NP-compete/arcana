@ui
Feature: FinOps cost management

  As an SRE
  I want to monitor AI spending and resource usage
  So that I can optimize costs and stay within budget

  Background:
    Given I am logged in as "admin"
    When I navigate to "/finops"

  @smoke
  Scenario: FinOps page loads with heading
    Then I should see the "FinOps" page

  Scenario: FinOps dashboard shows statistics cards
    Then I should see 4 statistics cards

  Scenario: FinOps displays cost charts
    Then I should see a chart or visualization

  Scenario: Cost breakdown table is visible
    Then I should see a table with at least 1 rows

  Scenario: Period selector is available
    When I select period "Last 30 days"
    Then I should see a chart or visualization
