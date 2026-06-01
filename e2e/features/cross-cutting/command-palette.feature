@ui
Feature: Command palette search

  As a power user
  I want to open a command palette with a keyboard shortcut
  So that I can quickly search and navigate to any feature

  Background:
    Given I am logged in as "admin"
    When I navigate to "/dashboard"

  @smoke
  Scenario: Open command palette and see search input
    When I click the "command-palette" button
    Then I should see "Search"

  Scenario: Search for a feature in the command palette
    When I click the "command-palette" button
    Then I should see "Search"
    When I click the "Agents" button
    Then I should see "Agents"
