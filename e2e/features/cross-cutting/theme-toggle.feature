@ui
Feature: Theme toggle between dark and light mode

  As a platform user
  I want to switch between dark and light themes
  So that I can use the interface in my preferred visual mode

  Background:
    Given I am logged in as "admin"

  @smoke
  Scenario: Toggle from dark mode to light mode
    When I navigate to "/dashboard"
    And I click the "theme-toggle" button
    Then I should see "light"

  Scenario: Toggle from light mode back to dark mode
    When I navigate to "/dashboard"
    And I click the "theme-toggle" button
    And I click the "theme-toggle" button
    Then I should see "dark"
