@api
Feature: Role-based API access control

  As a platform operator
  I want the API to enforce role-based access
  So that users cannot read or modify resources outside their authorization scope

  # ── Basic role identity ──

  @smoke
  Scenario Outline: Each role receives its own identity from whoami
    Given I set the API role to "<role>"
    When I send a GET request to "/api/v1/auth/whoami"
    Then the response should be successful
    And the response JSON at "user_id" should be "<user_id>"
    And the response JSON at "role" should be "<role>"

    Examples:
      | role          | user_id   |
      | admin         | anonymous |
      | developer     | alex      |
      | data-engineer | priya     |
      | sre           | jordan    |
      | auditor       | sam       |
      | user          | maya      |

  # ── User: restricted from enterprise features ──

  @smoke
  Scenario: User is denied access to enterprise configuration
    Given I set the API role to "user"
    When I send a GET request to "/api/v1/enterprise/config"
    Then the response status should be 403

  Scenario: User is denied access to audit logs
    Given I set the API role to "user"
    When I send a GET request to "/api/v1/audit/logs"
    Then the response status should be 403

  Scenario: User can access the agents listing
    Given I set the API role to "user"
    When I send a GET request to "/api/v1/agents"
    Then the response should be successful

  # ── Developer: builder access ──

  Scenario: Developer can list agents
    Given I set the API role to "developer"
    When I send a GET request to "/api/v1/agents"
    Then the response should be successful

  Scenario: Developer can access connectors
    Given I set the API role to "developer"
    When I send a GET request to "/api/v1/connectors"
    Then the response should be successful

  Scenario: Developer is denied access to audit logs
    Given I set the API role to "developer"
    When I send a GET request to "/api/v1/audit/logs"
    Then the response status should be 403

  Scenario: Developer is denied access to settings
    Given I set the API role to "developer"
    When I send a GET request to "/api/v1/settings"
    Then the response status should be 403

  # ── Auditor: read-only compliance access ──

  Scenario: Auditor can access audit logs
    Given I set the API role to "auditor"
    When I send a GET request to "/api/v1/audit/logs"
    Then the response should be successful

  Scenario: Auditor is denied access to enterprise configuration
    Given I set the API role to "auditor"
    When I send a GET request to "/api/v1/enterprise/config"
    Then the response status should be 403

  # ── SRE: operations access ──

  Scenario: SRE can access settings
    Given I set the API role to "sre"
    When I send a GET request to "/api/v1/settings"
    Then the response should be successful

  Scenario: SRE can access FinOps data
    Given I set the API role to "sre"
    When I send a GET request to "/api/v1/finops"
    Then the response should be successful

  Scenario: SRE is denied access to audit logs
    Given I set the API role to "sre"
    When I send a GET request to "/api/v1/audit/logs"
    Then the response status should be 403

  # ── Admin: unrestricted access ──

  @smoke
  Scenario Outline: Admin can access any endpoint
    Given I set the API role to "admin"
    When I send a GET request to "<endpoint>"
    Then the response should be successful

    Examples:
      | endpoint                   |
      | /api/v1/agents             |
      | /api/v1/connectors         |
      | /api/v1/enterprise/config  |
      | /api/v1/audit/logs         |
      | /api/v1/settings           |
      | /api/v1/finops             |
