@ui
Feature: Audit log explorer

  As an auditor
  I want to browse and filter the platform audit trail
  So that I can verify compliance and investigate security events

  Background:
    Given I am logged in as "admin"
    When I navigate to "/audit"

  @smoke
  Scenario: Audit page loads with heading
    Then I should see the "Audit" page

  Scenario: Audit log displays entries table
    Then I should see a table with at least 1 rows

  Scenario: Audit log shows filter options
    Then I should see "Filter"

  Scenario: Export button is available
    Then I should see the "Export" button

  Scenario: Audit page matches visual baseline
    Then the page should match the visual baseline "studio-audit-explorer"
