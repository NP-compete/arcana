@ui
Feature: Approval workflows

  As a platform administrator
  I want to review and approve pending requests
  So that I can enforce governance policies before changes go live

  Background:
    Given I am logged in as "admin"
    When I navigate to "/approvals"

  @smoke
  Scenario: Approvals page loads with heading
    Then I should see the "Approvals" page

  Scenario: Approvals page shows statistics cards
    Then I should see 3 statistics cards

  Scenario: Pending approvals table is visible
    Then I should see "Pending"

  Scenario: Review button is available
    Then I should see the "Review" button

  Scenario: Approvals page matches visual baseline
    Then the page should match the visual baseline "studio-approvals"
