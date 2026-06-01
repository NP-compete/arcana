@ui
Feature: Organization chart

  As an SRE
  I want to view the agent organization hierarchy
  So that I can understand team structure and agent relationships

  Background:
    Given I am logged in as "admin"
    When I navigate to "/org-chart"

  @smoke
  Scenario: Org chart page loads with heading
    Then I should see the "Org Chart" page

  Scenario: Org chart shows agent cards
    Then I should see "Agents"

  Scenario: Org chart displays summary statistics
    Then I should see 3 statistics cards

  Scenario: Team filter is available
    Then I should see "Team"

  Scenario: Org chart matches visual baseline
    Then the page should match the visual baseline "studio-org-chart"
