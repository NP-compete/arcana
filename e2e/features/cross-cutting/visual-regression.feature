@visual @ui
Feature: Visual regression baselines for key pages

  As a quality engineer
  I want to capture visual snapshots of critical pages
  So that unintended visual changes are detected before release

  @smoke
  Scenario Outline: <page_name> page matches visual baseline
    Given I am logged in as "admin"
    When I navigate to "<path>"
    Then the page should match the visual baseline "<baseline>"

    Examples:
      | page_name | path       | baseline            |
      | Login     | /login     | login-page          |
      | Dashboard | /dashboard | dashboard-page      |
      | Agents    | /agents    | agents-page         |
      | Settings  | /settings  | settings-page       |
