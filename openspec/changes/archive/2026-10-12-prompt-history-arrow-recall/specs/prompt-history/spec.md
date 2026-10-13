# Spec Delta

## ADDED Requirements

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
