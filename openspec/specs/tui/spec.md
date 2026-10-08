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

When the selected model offers a choice of aspect ratios, the form SHALL let the user open a selection dialog that lists all of the model's offered aspect ratios at once, with the current selection indicated. Within the dialog the user SHALL be able to move among the listed values, confirm one as the new selection, or cancel and leave the selection unchanged. The form SHALL show which value is currently selected and SHALL carry the selected value into the generation request for that model. On entering the form for such a model, the form SHALL default the selection to the last-used aspect ratio when that value is among the model's offered ratios, otherwise to the model's first offered ratio. When the user generates, the form SHALL record the selected aspect ratio as the last-used selection. A model that offers no aspect ratios SHALL present no aspect-ratio control.

When the selected model accepts reference images, the form SHALL let the user attach one or more images, either from the operating-system clipboard via a dedicated keypress or by entering a local file path on its own line in the prompt prefixed with a reserved sigil. At submit, the form SHALL extract sigil-prefixed path lines, resolve them (including leading `~` expansion), attach each resolved image, and remove those lines from the prompt text sent to the provider. The form SHALL display a hint naming the paste key, SHALL show an indicator of each attached image, and SHALL let the user remove attached images. Reading the clipboard or a referenced file SHALL be best-effort: a clipboard that holds no image, a path that is missing or unreadable, or data that cannot be decoded as a supported image SHALL produce a non-fatal status message, never an application error. Attached images SHALL be carried into the generation request for that model.

#### Scenario: Model offers aspect ratios

- **WHEN** the selected model offers a choice of aspect ratios
- **THEN** the form shows the currently selected value and a key hint for opening the aspect-ratio dialog

#### Scenario: Dialog lists all offered ratios

- **WHEN** the user opens the aspect-ratio dialog for a model that offers a choice of ratios
- **THEN** the dialog lists every offered ratio at once with the current selection indicated

#### Scenario: Select an aspect ratio

- **WHEN** the user confirms a value in the aspect-ratio dialog
- **THEN** the dialog closes, the form shows that value as selected, and it is included in the generation request for that model

#### Scenario: Cancel the dialog

- **WHEN** the user cancels the aspect-ratio dialog without confirming
- **THEN** the dialog closes and the previously selected value is left unchanged

#### Scenario: Default selection prefers last-used

- **WHEN** the user enters the form for a model that offers aspect ratios and the last-used aspect ratio is among the model's offered values
- **THEN** the form pre-selects the last-used aspect ratio

#### Scenario: Default selection falls back to first

- **WHEN** the user enters the form for a model that offers aspect ratios and there is no last-used aspect ratio, or the last-used value is not among the model's offered values
- **THEN** the form pre-selects the model's first offered aspect ratio

#### Scenario: Selected aspect ratio is remembered

- **WHEN** the user generates with an aspect ratio selected
- **THEN** that aspect ratio is recorded as the last-used selection

#### Scenario: Model offers no aspect ratios

- **WHEN** the selected model offers no aspect ratios
- **THEN** the form presents no aspect-ratio control

#### Scenario: Model supports reference images

- **WHEN** the selected model accepts reference images
- **THEN** the form offers an action to attach one or more reference images and shows a hint naming the key that pastes an image from the clipboard

#### Scenario: Model does not support a parameter

- **WHEN** the selected model does not support a given generation parameter
- **THEN** the form does not present a control for that parameter

#### Scenario: Switching models updates the form

- **WHEN** the user changes the selected model
- **THEN** the form's available controls update to match the new model's capabilities

#### Scenario: Paste action hidden for incapable model

- **WHEN** the selected model does not accept reference images
- **THEN** the form offers no reference-image attach action and no paste hint, and the paste keypress has no effect

#### Scenario: Paste an image from the clipboard

- **WHEN** the model accepts reference images and the user presses the paste key while the clipboard holds a usable image
- **THEN** the image is validated and attached to the draft, and the form shows an indicator for the newly attached image

#### Scenario: Multiple attached images are each indicated

- **WHEN** the user has attached more than one reference image
- **THEN** the form shows a distinct indicator for each attached image together with a visible count

#### Scenario: Remove an attached image

- **WHEN** the user invokes the remove action with at least one image attached
- **THEN** one attached image is removed and the indicator updates to reflect the remaining images, returning to the no-image state when none remain

#### Scenario: Clipboard holds no usable image

- **WHEN** the user presses the paste key and the clipboard holds no image or holds data that cannot be decoded as a supported image
- **THEN** the form shows a transient, non-fatal status message and attaches nothing

#### Scenario: Clipboard cannot be read

- **WHEN** the user presses the paste key but the clipboard is unavailable in the current environment (for example, no display, or over a remote session)
- **THEN** the form reports that no image could be read without erroring the application, and generation remains possible without a reference image

#### Scenario: Attached images are sent with the generation

- **WHEN** the user generates with one or more reference images attached to a reference-capable model
- **THEN** the attached images are included in the generation request for that model

#### Scenario: Attach by file path

- **WHEN** the model accepts reference images and the user submits a prompt containing a line that is a reserved sigil followed by a path to a readable, supported image
- **THEN** the image at that path is validated and attached, the sigil line is removed from the prompt text sent to the provider, and the form shows an indicator for the attached image

#### Scenario: File path is missing or unreadable

- **WHEN** the user submits a prompt whose sigil-prefixed path refers to a file that does not exist or cannot be read
- **THEN** the form shows a transient, non-fatal status message, attaches nothing for that path, and does not send the generation until the issue is addressed

#### Scenario: File path is not a usable image

- **WHEN** the user submits a prompt whose sigil-prefixed path refers to a file that cannot be decoded as a supported image or exceeds the size cap
- **THEN** the form shows a transient, non-fatal status message and attaches nothing for that path

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

### Requirement: Review step frames the prompt within a bounded box

At the review step, the system SHALL display the prompt inside a bordered box whose width is bounded by the current terminal width. Prompt text that is longer than the available width SHALL wrap within the box, and the box border SHALL fully enclose the wrapped text — no prompt text SHALL extend past the border, and the top and bottom edges SHALL align with the sides regardless of prompt length. The box SHALL reflow to the terminal width when the terminal is resized.

#### Scenario: Short prompt

- **WHEN** the review step is shown for a prompt that fits on one line
- **THEN** the prompt appears inside a bordered box whose border fully encloses it

#### Scenario: Long prompt wraps inside the box

- **WHEN** the review step is shown for a prompt longer than the available width
- **THEN** the prompt wraps onto multiple lines inside the box and no line extends past the right border

#### Scenario: Box reflows on resize

- **WHEN** the terminal is resized while the review step is shown
- **THEN** the box width adjusts to the new terminal width and the prompt re-wraps to fit, with the border still fully enclosing the text
