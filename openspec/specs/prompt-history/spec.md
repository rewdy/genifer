# prompt-history Specification

## Purpose

Persists a versioned record of every successful generation and lets users
review recent prompts newest-first and reuse one to pre-fill a new generation,
with a config option to turn the feature off.

## Requirements

### Requirement: Successful generations are recorded in history

When history is enabled and a generation succeeds, the system SHALL append a
record to an app-owned history store capturing at least the prompt text, the
model used, the aspect ratio, the seed used (whenever a seed was used), the
saved image path, and a timestamp. The seed SHALL be recorded so a past
generation can be reproduced or deliberately varied. A failed or cancelled
generation SHALL NOT be recorded.

#### Scenario: Successful generation is recorded
- **WHEN** history is enabled and a generation completes successfully
- **THEN** a new history record is persisted with the prompt, model, aspect ratio, saved path, timestamp, and the seed used

#### Scenario: Seed recorded for seed-capable generations
- **WHEN** a recorded generation used a seed
- **THEN** the record includes that seed value so the generation can be reproduced or varied later

#### Scenario: Failed generation is not recorded
- **WHEN** a generation fails or is cancelled
- **THEN** no history record is written

### Requirement: History storage is unbounded and resilient

The history store SHALL be an app-owned file in the config directory and SHALL
be versioned. It SHALL retain every recorded entry; records are never pruned or
deleted by the application. A missing, unparsable, or wrong-version history file
SHALL be treated as an empty history rather than an error, and the app SHALL
start normally.

#### Scenario: All entries are retained
- **WHEN** a new entry is recorded
- **THEN** it is added to the store and no previously recorded entry is removed

#### Scenario: Corrupt history file is non-fatal
- **WHEN** the history file is missing, unparsable, or an unrecognized version
- **THEN** the app starts with an empty history and does not error

### Requirement: Users can review and reuse past prompts

The TUI SHALL provide a history view that lists recorded entries newest-first.
To keep the view responsive, it SHALL load at most a bounded number of the
most-recent entries rather than the entire store. Selecting an entry SHALL
return the user to the compose screen pre-filled with that entry's prompt and
the parameters recoverable from the record (aspect ratio and seed where the
active model supports them).

When the user reaches the end of the loaded entries and more entries exist in
the store than were loaded, or when a filter matches none of the loaded
entries, the view SHALL point the user to the history file in the config
directory so they can find the full record there.

#### Scenario: History lists newest first
- **WHEN** the user opens the history view
- **THEN** recorded prompts are shown with the most recent entry first

#### Scenario: View loads a bounded number of recent entries
- **WHEN** the store holds more entries than the view's load limit
- **THEN** the view loads the most-recent entries up to that limit, newest-first

#### Scenario: Full history is pointed to when paging past the loaded entries
- **WHEN** the user pages to the end of the loaded entries and more entries exist in the store than were loaded
- **THEN** the view shows a message pointing to the history file in the config directory

#### Scenario: Empty filter result points to the full history
- **WHEN** the user filters the history view and no loaded entry matches
- **THEN** the view shows a message pointing to the history file in the config directory

#### Scenario: Selecting an entry pre-fills compose
- **WHEN** the user selects a history entry
- **THEN** the compose screen opens with that entry's prompt text, and its aspect ratio and seed applied where the selected model supports them

#### Scenario: Empty history is communicated
- **WHEN** the user opens the history view and no entries exist
- **THEN** the view indicates that there is no history yet

### Requirement: History can be disabled via config

History SHALL be enabled by default and SHALL be disableable through a config
option. When disabled, the system SHALL NOT write history records, and the
history view SHALL indicate that history is turned off.

#### Scenario: Default is enabled
- **WHEN** the config does not set the history option
- **THEN** history is enabled and successful generations are recorded

#### Scenario: Disabled stops recording
- **WHEN** history is disabled in config and a generation succeeds
- **THEN** no history record is written

#### Scenario: Disabled history view communicates its state
- **WHEN** history is disabled and the user opens the history view
- **THEN** the view indicates that history is turned off
