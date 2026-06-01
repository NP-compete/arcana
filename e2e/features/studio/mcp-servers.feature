@ui
Feature: MCP servers registry

  As a developer
  I want to view and search MCP servers
  So that I can discover available tool integrations for my agents

  Background:
    Given I am logged in as "admin"
    When I navigate to "/mcp"

  @smoke
  Scenario: MCP page loads with heading
    Then I should see the "MCP" page

  Scenario: MCP page displays server list
    Then I should see a table with at least 1 rows

  Scenario: Search filters MCP servers
    When I type "filesystem" in the search box
    Then I should see "filesystem"

  Scenario: MCP page shows tool details on expand
    When I expand the first table row
    Then I should see "Tools"

  Scenario: MCP page matches visual baseline
    Then the page should match the visual baseline "studio-mcp-servers"
