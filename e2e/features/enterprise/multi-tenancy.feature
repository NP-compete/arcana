@api
Feature: Multi-tenancy management

  As a platform administrator
  I want to create and manage tenants
  So that the platform can serve multiple isolated organizations

  Background:
    Given I set the API role to "admin"

  @smoke
  Scenario: Create a new tenant
    When I create a tenant with ID "tenant-alpha" and name "Alpha Corp"
    Then the response should be successful
    And the tenant "tenant-alpha" should exist

  Scenario: Create a second tenant
    When I create a tenant with ID "tenant-beta" and name "Beta Industries"
    Then the response should be successful
    And the tenant "tenant-beta" should exist

  @smoke
  Scenario: List all tenants returns created tenants
    When I list all tenants
    Then the response should be successful
    And the response should contain "tenant-alpha"
    And the response should contain "tenant-beta"

  Scenario: Create tenant via raw POST request
    When I send a POST request to "/api/v1/tenants" with body:
      """
      {
        "id": "tenant-gamma",
        "name": "Gamma LLC"
      }
      """
    Then the response should be successful
    And the response should contain "tenant-gamma"

  Scenario: List tenants via GET endpoint
    When I send a GET request to "/api/v1/tenants"
    Then the response should be successful
    And the response JSON should have property "tenants"
