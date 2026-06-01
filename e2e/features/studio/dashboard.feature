@ui @smoke
Feature: Studio dashboard

  As a platform administrator
  I want to see an overview of my Arcana platform
  So that I can quickly assess system status and take action

  Background:
    Given I am logged in as "admin"
    When I navigate to "/dashboard"

  Scenario: Dashboard page loads with heading
    Then I should see the "Dashboard" page
    And the page should have a nav sidebar

  Scenario: Dashboard displays statistics cards
    Then I should see 4 statistics cards

  Scenario: Dashboard shows quick-action cards
    Then I should see "Agents"
    And I should see "Models"
    And I should see "Connectors"

  Scenario: Admin sees the Deploy button on dashboard
    Then I should see the "Deploy" button

  Scenario: Dashboard matches visual baseline
    Then the page should match the visual baseline "studio-dashboard"
