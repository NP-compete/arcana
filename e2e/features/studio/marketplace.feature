@ui
Feature: Agent marketplace

  As a platform user
  I want to browse and deploy pre-built agents from the marketplace
  So that I can quickly adopt proven agent configurations

  Background:
    Given I am logged in as "admin"
    When I navigate to "/marketplace"

  @smoke
  Scenario: Marketplace page loads with heading
    Then I should see the "Marketplace" page

  Scenario: Marketplace displays available items
    Then I should see a table with at least 1 rows

  Scenario: Search filters marketplace items
    When I type "assistant" in the search box
    Then I should see "assistant"

  Scenario: Deploy button is visible for marketplace items
    Then I should see the "Deploy" button

  Scenario: Marketplace matches visual baseline
    Then the page should match the visual baseline "studio-marketplace"
