@ui @smoke
Feature: Platform settings

  As a platform administrator
  I want to configure platform-wide settings
  So that I can manage security, tenants, compliance, and access policies

  Background:
    Given I am logged in as "admin"
    When I navigate to "/settings"

  Scenario: Settings page loads with heading
    Then I should see the "Settings" page

  Scenario: Platform tab is active by default
    Then the "Platform" tab should be active

  Scenario: RBAC tab loads
    When I open the "RBAC" tab
    Then the "RBAC" tab should be active

  Scenario: Security tab loads
    When I open the "Security" tab
    Then the "Security" tab should be active

  Scenario: Settings page shows all six tabs
    Then I should see "Platform"
    And I should see "RBAC"
    And I should see "Security"
    And I should see "Tenants"
    And I should see "Audit"
    And I should see "Compliance"
