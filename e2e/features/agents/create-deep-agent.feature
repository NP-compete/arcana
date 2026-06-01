@api @slow
Feature: Create deep agent via chat flow

  As a developer
  I want to create a deep agent through the conversational chat interface
  So that I can deploy an agent with advanced capabilities like world model and skill graph

  Background:
    Given I set the API role to "admin"

  @smoke
  Scenario: Create a deep agent by selecting type 2
    When I start creating an agent via chat with name "bdd-deep-agent"
    Then the agent reply should contain "type"

    When I respond with "2" in the chat
    Then the agent reply should contain "connector"

    When I respond with "none" in the chat
    Then the agent reply should contain "budget"

    When I respond with "25" in the chat
    Then the agent reply should contain "confirm"

    When I respond with "yes" in the chat
    Then the agent reply should contain "created"
    And the agent "bdd-deep-agent" should be active
    And the agent "bdd-deep-agent" should have type "deep"

  Scenario: Deep agent type is persisted correctly
    Given an agent named "bdd-deep-verify" exists
    When I send a GET request to "/api/v1/agents"
    Then the response should be successful
    And the response should contain "bdd-deep-verify"

  Scenario: Deep agent properties are returned via API
    Given an agent named "bdd-deep-props" exists
    When I send a GET request to "/api/v1/agents"
    Then the response should be successful
    And the response JSON should have property "agents"
