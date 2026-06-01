@ui
Feature: Connectors management

  As a data engineer
  I want to browse and add data connectors
  So that I can integrate external data sources with the platform

  Background:
    Given I am logged in as "admin"
    When I navigate to "/connectors"

  @smoke
  Scenario: Connectors page loads with heading
    Then I should see the "Connectors" page

  Scenario: Connectors page shows connector type grid
    Then I should see "Database"
    And I should see "API"

  Scenario: Add connector button is visible
    Then I should see the "Add Connector" button

  Scenario: Connectors page matches visual baseline
    Then the page should match the visual baseline "studio-connectors"
