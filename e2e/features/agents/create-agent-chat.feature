@api @slow
Feature: Create standard agent via chat flow

  As a developer
  I want to create a standard agent through the conversational chat interface
  So that I can deploy an AI agent without navigating complex forms

  Background:
    Given I set the API role to "admin"

  @smoke
  Scenario: Create a standard agent through the full chat flow
    When I start creating an agent via chat with name "bdd-standard-agent"
    Then the agent reply should contain "type"

    When I respond with "1" in the chat
    Then the agent reply should contain "connector"

    When I respond with "none" in the chat
    Then the agent reply should contain "budget"

    When I respond with "10" in the chat
    Then the agent reply should contain "confirm"

    When I respond with "yes" in the chat
    Then the agent reply should contain "created"
    And the agent "bdd-standard-agent" should be active
    And the agent "bdd-standard-agent" should have type "standard"

  Scenario: Chat flow prompts for agent type after name
    When I start creating an agent via chat with name "bdd-type-prompt-test"
    Then the agent reply should contain "type"

  Scenario: Chat flow prompts for connectors after type selection
    When I start creating an agent via chat with name "bdd-connector-prompt-test"
    And I respond with "1" in the chat
    Then the agent reply should contain "connector"

  Scenario: Chat flow prompts for budget after connectors
    When I start creating an agent via chat with name "bdd-budget-prompt-test"
    And I respond with "1" in the chat
    And I respond with "none" in the chat
    Then the agent reply should contain "budget"

  Scenario: Verify created agent appears in agent list via API
    Given an agent named "bdd-listed-agent" exists
    When I send a GET request to "/api/v1/agents"
    Then the response should be successful
    And the response should contain "bdd-listed-agent"
