@api
Feature: Agent chat interactions

  As a developer
  I want to send messages to a running agent
  So that I can verify it responds with correct capabilities and handles tasks

  Background:
    Given I set the API role to "admin"
    And an agent named "bdd-chat-agent" exists

  @smoke
  Scenario: Agent responds with its capabilities
    When I send "What can you do?" to agent "bdd-chat-agent"
    Then the agent reply should contain "capabilities"

  Scenario: Agent handles a task submission
    When I send "Summarize the latest deployment logs" to agent "bdd-chat-agent"
    Then the chat response should have steps

  Scenario: Agent responds to a cost check
    When I send "How much budget do I have left?" to agent "bdd-chat-agent"
    Then the agent reply should contain "budget"
