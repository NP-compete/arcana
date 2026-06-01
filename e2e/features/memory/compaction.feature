@api @smoke
Feature: Memory compaction from short-term to long-term

  As a platform operator
  I want to compact short-term memory into long-term storage
  So that ephemeral context is consolidated into durable, searchable knowledge

  Background:
    Given an agent named "mem-compact-bdd" exists

  # ── Basic compaction ──

  Scenario: Compact short-term entries into long-term memory
    When I store short-term memory for agent "mem-compact-bdd" with key "fact_1" and value "User prefers Python over Go"
    And I store short-term memory for agent "mem-compact-bdd" with key "fact_2" and value "User is on the data platform team"
    And I compact memory for agent "mem-compact-bdd"
    Then the response should be successful
    And the compaction should report at least 2 compacted entry
    And a summary ID should be returned

  # ── Post-compaction state ──

  Scenario: Short-term memory is empty after compaction
    When I store short-term memory for agent "mem-compact-bdd" with key "temp_note" and value "Remind me about the standup"
    And I compact memory for agent "mem-compact-bdd"
    Then the compaction should report at least 1 compacted entry
    And the short-term memory for agent "mem-compact-bdd" should have 0 entries

  Scenario: Compacted content is searchable in long-term memory
    When I store short-term memory for agent "mem-compact-bdd" with key "pref_lang" and value "User speaks French and English"
    And I compact memory for agent "mem-compact-bdd"
    And I search long-term memory for agent "mem-compact-bdd" with query "spoken languages"
    Then the search results should have at least 1 entries

  # ── Idempotent compaction ──

  Scenario: Compacting with no short-term entries is a no-op
    When I compact memory for agent "mem-compact-bdd"
    Then the response should be successful
    And the compaction should report at least 0 compacted entry
