@ui
Feature: YAML editor

  As a developer
  I want to edit agent and resource configurations in YAML
  So that I can make precise adjustments using a familiar text format

  Background:
    Given I am logged in as "admin"
    When I navigate to "/editor"

  @smoke
  Scenario: Editor page loads with heading
    Then I should see the "Editor" page

  Scenario: Editor displays a text area
    Then I should see "apiVersion"

  Scenario: Editor shows save button
    Then I should see the "Save" button

  Scenario: Editor page matches visual baseline
    Then the page should match the visual baseline "studio-yaml-editor"
