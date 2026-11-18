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

### Requirement: Inline Up/Down history recall on the compose prompt

The compose prompt SHALL let the user recall recorded prompts inline with the
Up and Down arrow keys, scoped to the compose prompt only. Recall SHALL be
edge-aware so it never prevents normal cursor movement within a multi-line
prompt:

- Up SHALL recall the previous (older) entry only when the cursor is on the
  first line of the prompt; otherwise Up SHALL move the cursor within the text.
- Down SHALL advance to the next (newer) entry only when the cursor is on the
  last line of the prompt; otherwise Down SHALL move the cursor within the text.

Recall SHALL draw from the same most-recent loaded entries as the history view.
When an entry is recalled, its aspect ratio and seed SHALL be applied where the
active model supports them, consistent with selecting the entry from the history
view. Recall SHALL be available only on the compose screen and SHALL have no
effect in the picker, review, aspect-ratio dialog, or history view.

#### Scenario: Up at the first line recalls the previous entry
- **WHEN** the cursor is on the first line of the compose prompt and the user presses Up
- **THEN** the prompt is replaced with the previous (older) recorded entry

#### Scenario: Up away from the first line moves the cursor
- **WHEN** the cursor is not on the first line of a multi-line prompt and the user presses Up
- **THEN** the cursor moves up within the text and no history entry is recalled

#### Scenario: Down away from the last line moves the cursor
- **WHEN** the cursor is not on the last line of a multi-line prompt and the user presses Down
- **THEN** the cursor moves down within the text and no history entry is recalled

#### Scenario: Recall is scoped to compose
- **WHEN** the user presses Up or Down outside the compose prompt (for example in the picker or history view)
- **THEN** no inline history recall occurs and the key retains its usual behavior for that screen

### Requirement: Recall preserves the in-progress draft

Inline recall SHALL preserve the text the user had entered before recall began.
The system SHALL snapshot the in-progress prompt when a recall session starts
and SHALL restore it when the user steps forward (Down) past the newest entry,
so the user always returns to exactly what they were typing — including an empty
prompt. Recall SHALL never discard unsent text without a way to restore it.

#### Scenario: Stepping past the newest restores the draft
- **WHEN** the user has recalled one or more entries and presses Down past the newest entry
- **THEN** the prompt is restored to the in-progress text that was present when recall started

#### Scenario: Empty draft is restored
- **WHEN** recall started from an empty prompt and the user steps Down past the newest entry
- **THEN** the prompt returns to empty

### Requirement: Editing the prompt ends the recall session

A manual edit to the prompt during a recall session SHALL end that session, so
that a subsequent Up begins a new recall from the newest entry rather than
continuing from the previous position.

#### Scenario: Edit resets recall position
- **WHEN** the user recalls an entry, edits the prompt text, and then presses Up at the first line
- **THEN** recall begins again from the newest entry

### Requirement: History captures and restores generation settings

In addition to the prompt, model, aspect ratio, and seed, each successful history record SHALL capture the generation settings needed to reproduce output shape: the background (transparency) setting and the output format (including animated GIF). These settings SHALL be stored as optional fields so records written before this capability remain readable with empty values. When a user selects a history entry or recalls one inline with Up/Down, the system SHALL restore its settings alongside the prompt, applying each only where the currently selected model supports it. A setting a model does not support SHALL be ignored in favor of that model's default, and the recalled GIF and transparent-background settings SHALL remain mutually exclusive. Starting a recall session SHALL snapshot the in-progress settings as well as the prompt, and stepping forward past the newest entry SHALL restore that snapshot together with the original prompt text.

#### Scenario: Generation settings are recorded

- **WHEN** a generation succeeds with transparency or GIF output selected
- **THEN** the history record includes the background and output-format settings used

#### Scenario: Older records remain readable

- **WHEN** a history record predates settings capture and has no background or output-format values
- **THEN** the record is read normally and recall falls back to the current model's defaults

#### Scenario: Selecting an entry restores settings

- **WHEN** the user selects a history entry recorded with GIF output on a model that supports animated GIF
- **THEN** compose is pre-filled with the prompt and animated GIF output is on

#### Scenario: Inline recall restores settings

- **WHEN** the user recalls a history entry with a transparent background on a model that supports transparency
- **THEN** the compose form shows the transparent background setting enabled alongside the recalled prompt

#### Scenario: Unsupported settings are not applied

- **WHEN** a recalled entry carries a setting that the currently selected model does not support
- **THEN** that setting is ignored and the model's default applies without blocking recall

#### Scenario: Conflicting settings resolve to one mode

- **WHEN** a recalled entry contains both transparent background and GIF output
- **THEN** the restored form enables animated GIF and leaves transparent background off

#### Scenario: Stepping past the newest restores the original settings

- **WHEN** the user recalls one or more entries and then steps forward past the newest entry
- **THEN** the original prompt text and its original background and output-format settings are restored
