@ui @smoke
Feature: Login and session management

  As a platform user
  I want to sign in with my assigned role
  So that I can access the features permitted for my persona

  Background:
    Given I am not logged in

  Scenario: Login page displays all six role cards
    Then I should be on the login page
    And the login page should show 6 role cards

  Scenario Outline: Sign in as each persona and verify identity
    Given I am logged in as "<role>"
    When I navigate to "/dashboard"
    Then I should see "<display_name>"

    Examples:
      | role          | display_name  |
      | admin         | Administrator |
      | developer     | Developer     |
      | data-engineer | Data Engineer |
      | sre           | SRE           |
      | auditor       | Auditor       |
      | user          | User          |

  Scenario Outline: Sign out returns to login page
    Given I am logged in as "<role>"
    When I sign out
    Then I should be on the login page
    And the login page should show 6 role cards

    Examples:
      | role      |
      | admin     |
      | developer |
      | user      |
