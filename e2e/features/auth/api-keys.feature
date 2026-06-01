@api
Feature: API key management

  As a platform administrator
  I want to manage API keys for programmatic access
  So that services and automation can authenticate without interactive login

  Background:
    Given I set the API role to "admin"

  @smoke
  Scenario: List API keys returns a successful response
    When I send a GET request to "/api/v1/api-keys"
    Then the response should be successful

  Scenario: Create a new API key
    When I send a POST request to "/api/v1/api-keys" with body:
      """
      {
        "name": "bdd-test-key",
        "description": "Key created by BDD test suite",
        "role": "developer"
      }
      """
    Then the response should be successful
    And the response JSON should have property "name"
    And the response JSON should have property "key"
    And the response JSON should have property "role"
    And the response JSON at "name" should be "bdd-test-key"
    And the response JSON at "role" should be "developer"

  Scenario: Created API key contains required metadata fields
    When I send a POST request to "/api/v1/api-keys" with body:
      """
      {
        "name": "bdd-metadata-key",
        "description": "Verify metadata fields",
        "role": "user"
      }
      """
    Then the response should be successful
    And the response JSON should have property "id"
    And the response JSON should have property "name"
    And the response JSON should have property "key"
    And the response JSON should have property "created_at"

  Scenario: Non-admin role is denied API key creation
    Given I set the API role to "developer"
    When I send a POST request to "/api/v1/api-keys" with body:
      """
      {
        "name": "unauthorized-key",
        "description": "Should be rejected",
        "role": "user"
      }
      """
    Then the response status should be 403

  Scenario: Non-admin role is denied API key listing
    Given I set the API role to "user"
    When I send a GET request to "/api/v1/api-keys"
    Then the response status should be 403
