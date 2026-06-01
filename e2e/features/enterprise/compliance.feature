@api
Feature: Compliance framework reporting

  As a compliance officer
  I want to retrieve compliance reports for industry-standard frameworks
  So that I can verify the platform meets regulatory requirements

  Background:
    Given I set the API role to "admin"

  @smoke
  Scenario Outline: Retrieve compliance report for <framework>
    When I check compliance for framework "<framework>"
    Then the response should be successful
    And the compliance report should have framework "<framework>"
    And the compliance report should have controls

    Examples:
      | framework |
      | soc2      |
      | gdpr      |
      | hipaa     |
      | iso27001  |
      | euaiact   |

  Scenario Outline: Compliance endpoint returns report via direct GET for <framework>
    When I send a GET request to "/api/v1/compliance?framework=<framework>"
    Then the response should be successful
    And the response should contain "<framework>"
    And the response JSON should have property "controls"

    Examples:
      | framework |
      | soc2      |
      | gdpr      |
      | hipaa     |
      | iso27001  |
      | euaiact   |
