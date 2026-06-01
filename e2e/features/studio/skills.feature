@ui
Feature: Skills registry

  As a developer
  I want to view and register agent skills
  So that I can extend agent capabilities with reusable skill modules

  Background:
    Given I am logged in as "admin"
    When I navigate to "/skills"

  @smoke
  Scenario: Skills page loads with heading
    Then I should see the "Skills" page

  Scenario: Register skill button is visible
    Then I should see the "Register Skill" button

  Scenario: Skills page shows tier architecture
    Then I should see "Tier"

  Scenario: Skills page displays skill cards
    Then I should see 3 statistics cards

  Scenario: Skills page matches visual baseline
    Then the page should match the visual baseline "studio-skills"
