@api
Feature: Audit logging and enterprise configuration

  As a platform administrator
  I want to review audit trails and verify enterprise configuration
  So that I can ensure operational integrity and proper security settings

  Background:
    Given I set the API role to "admin"

  # -- Audit log entries --

  @smoke
  Scenario: Fetch audit log entries
    When I fetch audit entries
    Then the response should be successful
    And the audit log should have at least 1 entries

  Scenario: Audit entries endpoint returns data via GET
    When I send a GET request to "/api/v1/enterprise/audit"
    Then the response should be successful
    And the response JSON should have property "entries"

  # -- Audit chain integrity --

  @smoke
  Scenario: Audit chain integrity is valid
    When I send a GET request to "/api/v1/enterprise/audit/stats"
    Then the response should be successful
    And the audit chain should be intact

  Scenario: Audit stats returns chain_valid field
    When I send a GET request to "/api/v1/enterprise/audit/stats"
    Then the response should be successful
    And the response JSON at "chain_valid" should be "true"

  # -- Enterprise configuration --

  @smoke
  Scenario: Enterprise config returns security settings
    When I send a GET request to "/api/v1/enterprise/config"
    Then the response should be successful
    And the response JSON should have property "auth_mode"
    And the response JSON should have property "rbac_enabled"
    And the response JSON should have property "rate_limiting_enabled"
    And the response JSON should have property "audit_enabled"

  Scenario: Enterprise config includes tenancy and compliance settings
    When I send a GET request to "/api/v1/enterprise/config"
    Then the response should be successful
    And the response JSON should have property "multi_tenancy"
    And the response JSON should have property "compliance_frameworks"
    And the response JSON should have property "roles"
