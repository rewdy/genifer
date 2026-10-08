# tui Specification

## Purpose

Defines the full-screen terminal interface shell that drives genifer — its header, model picker, capability-adaptive generate form, status display, and keybindings.

## Requirements

### Requirement: Full-screen application shell

The system SHALL run as a full-screen terminal application with a visually distinct header displayed prominently, and SHALL restore the terminal to its prior state on exit.

#### Scenario: Launch

- **WHEN** genifer is started in a capable terminal
- **THEN** it takes over the full screen and displays the styled header

#### Scenario: Clean exit

- **WHEN** the user quits genifer
- **THEN** the terminal is restored to its normal state with no residual full-screen artifacts

#### Scenario: Resize

- **WHEN** the terminal is resized
- **THEN** the layout reflows to the new dimensions without corrupting the display

### Requirement: Model selection

The system SHALL present a model picker populated from the provider's model list, SHALL pre-select the last-used model when it is still available, and SHALL persist the chosen model as the last-used selection.

#### Scenario: Choose a model

- **WHEN** the user selects a model from the picker
- **THEN** that model becomes active for generation and is recorded as the last-used selection

#### Scenario: Pre-selected on launch

- **WHEN** a last-used model exists and is still offered by the provider
- **THEN** it is pre-selected when the picker is shown

#### Scenario: Model list unavailable

- **WHEN** the model list cannot be loaded
- **THEN** the TUI shows an actionable error (distinguishing authentication problems) instead of an empty or broken picker

### Requirement: Capability-adaptive generate form

The generate form SHALL reflect the selected model's capabilities: controls for parameters the model does not support SHALL be hidden or disabled, and the option to attach reference images SHALL be available only when the model accepts them.

#### Scenario: Model supports reference images

- **WHEN** the selected model accepts reference images
- **THEN** the form offers an action to attach one or more reference images

#### Scenario: Model does not support a parameter

- **WHEN** the selected model does not support a given generation parameter
- **THEN** the form does not present a control for that parameter

#### Scenario: Switching models updates the form

- **WHEN** the user changes the selected model
- **THEN** the form's available controls update to match the new model's capabilities

### Requirement: Scrollable, filterable model list

The model picker SHALL show at most a bounded number of models at once and SHALL scroll when the list is longer, so the header remains visible regardless of how many models the provider offers. The picker SHALL provide a filter that narrows the visible models by a typed query matching the model name or identifier.

#### Scenario: Long list does not push the header off-screen

- **WHEN** the provider returns more models than fit in the visible area
- **THEN** the picker shows a capped window of models and scrolls through the rest while the header stays visible

#### Scenario: Filter by query

- **WHEN** the user activates the filter and types a query
- **THEN** the picker shows only models whose name or identifier matches the query

#### Scenario: Each model occupies a single row

- **WHEN** the picker is displayed
- **THEN** each model is shown on one line, with its name and any secondary detail on that same line

### Requirement: Per-model pricing display

The picker SHALL show each model's output-image price alongside its name, classified compactly as a per-image price, a per-token indication, free, or omitted when unknown. Pricing SHALL be presented as secondary detail, visually distinct from the model name, and SHALL never block interaction: the picker is usable before pricing resolves.

#### Scenario: Per-image price shown

- **WHEN** a model is priced per image and its price has resolved
- **THEN** the picker shows a compact per-image price (e.g. `$0.018/img`) next to the model name

#### Scenario: Non per-image pricing

- **WHEN** a model is priced per token
- **THEN** the picker shows a compact per-token indication rather than a dollar-per-image figure

#### Scenario: Pricing still loading

- **WHEN** a model's price has not yet resolved
- **THEN** the picker shows a loading indicator for that model and remains fully interactive

#### Scenario: Pricing unavailable

- **WHEN** a model's price cannot be determined
- **THEN** the picker omits a price for that model rather than showing a misleading value

#### Scenario: Detail is visually secondary

- **WHEN** price and capability detail are shown next to a model name
- **THEN** that detail is rendered in a muted style distinct from the model name

### Requirement: Status and keybindings

The system SHALL display current status (idle, generating, saved, error) and SHALL provide discoverable keybindings for the primary actions, including submitting a generation, cancelling an in-flight generation, opening a saved image, and quitting.

#### Scenario: Keybinding hints visible

- **WHEN** the TUI is displayed
- **THEN** the available key actions for the current state are shown

#### Scenario: Status reflects generation lifecycle

- **WHEN** the generation lifecycle moves between idle, in-flight, saved, and error
- **THEN** the displayed status updates to match the current state

### Requirement: First-run onboarding flow

When genifer launches and no `config.yaml` exists, the TUI SHALL present a first-run onboarding flow before the normal shell: a prominent WELCOME banner and a prompt to choose how the OpenRouter API key will be provided. The flow SHALL offer three choices — paste a literal key, reference an environment variable, or run a command — and SHALL write the resulting value into a newly created `config.yaml` as the matching directive before continuing into the normal session. The onboarding flow SHALL run only when no config file exists; when a config file is present, genifer SHALL skip onboarding and go straight to the normal shell.

#### Scenario: Onboarding shown on first run

- **WHEN** genifer launches and no `config.yaml` exists
- **THEN** the TUI displays a WELCOME banner and the API-key provider prompt instead of starting directly in the model picker

#### Scenario: Onboarding skipped when config exists

- **WHEN** genifer launches and `config.yaml` exists
- **THEN** the onboarding flow is not shown and the normal shell starts

#### Scenario: Choose environment variable

- **WHEN** the user chooses to provide the key via an environment variable and supplies the variable name
- **THEN** genifer writes the API key as `{env:NAME}` into the newly created `config.yaml` and continues into the normal session

#### Scenario: Choose command

- **WHEN** the user chooses to provide the key via a command and supplies the command
- **THEN** genifer writes the API key as `{cmd:...}` into the newly created `config.yaml` and continues into the normal session

#### Scenario: Choose paste with warning

- **WHEN** the user chooses to paste a literal key
- **THEN** genifer warns that the key will be stored as plain text in `config.yaml`, and on confirmation writes the pasted key as a literal value and continues

#### Scenario: Writing the config honors create-only ownership

- **WHEN** onboarding completes and writes `config.yaml`
- **THEN** it creates the file only because none existed, writing a commented starter populated with the chosen API-key directive, and never overwrites an existing config
