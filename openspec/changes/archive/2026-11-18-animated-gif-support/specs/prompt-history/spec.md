# Spec Delta

## ADDED Requirements

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
