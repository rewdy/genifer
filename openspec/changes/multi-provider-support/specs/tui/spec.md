# Spec Delta

## MODIFIED Requirements

### Requirement: Model selection

The system SHALL present a model picker populated by merging the model lists of
every configured provider, with each model attributed to the provider that
offers it. A model SHALL be identified by the combination of its provider and
its model id, so two providers offering the same model id do not collide. The
picker SHALL group or otherwise indicate each model's provider. The system SHALL
pre-select the last-used model — matched by provider and model id — when it is
still available, and SHALL persist the chosen model, together with its provider,
as the last-used selection. When the user generates, the generation SHALL be
routed to the provider that owns the selected model.

A provider that cannot be reached while the picker is being populated SHALL be
isolated: its models are omitted and it is shown as offline, while the models of
every reachable provider remain listed and usable. One unreachable provider
SHALL NOT blank the picker or error the application.

#### Scenario: Choose a model

- **WHEN** the user selects a model from the picker
- **THEN** that model becomes active for generation, generation routes to the provider that owns it, and the model and its provider are recorded as the last-used selection

#### Scenario: Models merged across providers

- **WHEN** more than one provider is configured and reachable
- **THEN** the picker lists the models of every reachable provider, each indicating which provider offers it

#### Scenario: Same model id across providers

- **WHEN** two configured providers each offer a model with the same identifier
- **THEN** the picker lists both as distinct entries attributed to their respective providers, and selecting one targets that provider

#### Scenario: Pre-selected on launch

- **WHEN** a last-used model and provider exist and that model is still offered by that provider
- **THEN** it is pre-selected when the picker is shown

#### Scenario: Unreachable provider isolated

- **WHEN** one configured provider cannot be reached while the picker is populated and at least one other provider is reachable
- **THEN** the unreachable provider is shown as offline, its models are omitted, and the reachable providers' models remain listed and usable

#### Scenario: Model list unavailable

- **WHEN** no provider's model list can be loaded
- **THEN** the TUI shows an actionable error (distinguishing authentication problems) instead of an empty or broken picker

### Requirement: Per-model pricing display

The picker SHALL show each model's output-image price alongside its name,
classified compactly as a per-image price, a per-token indication, free, or
omitted when unknown. Prices SHALL be resolved and displayed per provider
instance, so a model offered by two providers shows each provider's own price.
Pricing SHALL be presented as secondary detail, visually distinct from the model
name, and SHALL never block interaction: the picker is usable before pricing
resolves.

#### Scenario: Per-image price shown

- **WHEN** a model is priced per image and its price has resolved
- **THEN** the picker shows a compact per-image price (e.g. `$0.018/img`) next to the model name

#### Scenario: Free price shown

- **WHEN** a model's provider reports it as free and the price has resolved
- **THEN** the picker shows a compact free indication next to the model name

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

### Requirement: First-run onboarding flow

When genifer launches and no usable `config.yaml` exists — either because none
is present or because a legacy-shape file was detected and backed up — the TUI
SHALL present a first-run onboarding flow before the normal shell: a prominent
WELCOME banner and a prompt to configure a provider. The flow SHALL first let
the user choose the provider type (a hosted OpenRouter provider or a local
provider), then collect the detail that type requires:

- For OpenRouter: how the API key will be provided — paste a literal key,
  reference an environment variable, or run a command.
- For the local provider: the address of the running WebUI, with a sensible
  default offered.

The flow SHALL write the result into a newly created `config.yaml` as a single
provider instance in the provider-list shape before continuing into the normal
session. The onboarding flow SHALL run only when no usable config file exists;
when a valid current-shape config is present, genifer SHALL skip onboarding and
go straight to the normal shell.

#### Scenario: Onboarding shown on first run

- **WHEN** genifer launches and no `config.yaml` exists
- **THEN** the TUI displays a WELCOME banner and the provider-type prompt instead of starting directly in the model picker

#### Scenario: Onboarding shown after legacy recovery

- **WHEN** genifer launches, detects a legacy-shape `config.yaml`, and backs it up
- **THEN** the TUI enters the onboarding flow to rebuild configuration rather than starting in the model picker

#### Scenario: Onboarding skipped when config exists

- **WHEN** genifer launches and a valid current-shape `config.yaml` exists
- **THEN** the onboarding flow is not shown and the normal shell starts

#### Scenario: Choose environment variable

- **WHEN** the user chooses the OpenRouter provider type and chooses to provide the key via an environment variable, supplying the variable name
- **THEN** genifer writes a single OpenRouter provider instance whose API key is `{env:NAME}` into the newly created `config.yaml` and continues into the normal session

#### Scenario: Choose command

- **WHEN** the user chooses the OpenRouter provider type and chooses to provide the key via a command, supplying the command
- **THEN** genifer writes a single OpenRouter provider instance whose API key is `{cmd:...}` into the newly created `config.yaml` and continues into the normal session

#### Scenario: Choose paste with warning

- **WHEN** the user chooses the OpenRouter provider type and chooses to paste a literal key
- **THEN** genifer warns that the key will be stored as plain text, and on confirmation writes a single OpenRouter instance with the pasted literal key and continues

#### Scenario: Choose the local provider

- **WHEN** the user chooses the local provider type and accepts or edits the WebUI address
- **THEN** genifer writes a single local provider instance with that address into the newly created `config.yaml` and continues into the normal session

#### Scenario: Writing the config honors create-only ownership

- **WHEN** onboarding completes and writes `config.yaml`
- **THEN** it writes a commented starter in the provider-list shape populated with the chosen provider instance, and never overwrites a valid existing config
