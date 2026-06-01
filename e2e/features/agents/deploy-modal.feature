@ui
Feature: Agent deploy modal

  As a developer
  I want to open the deploy modal from the Agents page
  So that I can create an agent using a visual form instead of the chat flow

  Background:
    Given I am logged in as "admin"
    When I navigate to "/agents"

  @smoke
  Scenario: Deploy button is visible on the Agents page
    Then I should see the "Deploy" button

  Scenario: Deploy modal opens when clicking Deploy
    When I click the "Deploy" button
    Then I should see "Deploy Agent"

  Scenario: Deploy modal shows agent name field
    When I click the "Deploy" button
    Then I should see "Name"

  Scenario: Deploy modal shows agent type selection
    When I click the "Deploy" button
    Then I should see "Type"
