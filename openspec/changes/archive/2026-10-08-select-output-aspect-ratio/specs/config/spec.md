# Spec Delta

## MODIFIED Requirements

### Requirement: State persistence separate from config

The system SHALL persist last-used selections (at minimum, the last selected model and the last selected aspect ratio) in a separate app-written file (`state.json` in the same config directory). The system SHALL write to `state.json` only, never to `config.yaml`. When persisting one selection, the system SHALL preserve the other previously persisted selections rather than overwriting them.

#### Scenario: Remember last model

- **WHEN** the user selects a model and generates, then restarts genifer
- **THEN** the previously selected model is pre-selected on next launch

#### Scenario: Remember last aspect ratio

- **WHEN** the user generates with an aspect ratio selected, then restarts genifer
- **THEN** the previously selected aspect ratio is the remembered last-used value and is preferred as the default when a model offers it

#### Scenario: Persisting one selection preserves the other

- **WHEN** the user changes the model after an aspect ratio has already been remembered
- **THEN** the remembered aspect ratio is retained in `state.json` and not cleared by persisting the new model

#### Scenario: State file missing or invalid

- **WHEN** `state.json` is absent or cannot be parsed
- **THEN** genifer starts without a remembered selection and continues normally, recreating the file on next write

#### Scenario: Remembered model no longer available

- **WHEN** the remembered model is not present in the current provider model list
- **THEN** genifer does not pre-select it, falls back to no selection (or a default), and does not error
