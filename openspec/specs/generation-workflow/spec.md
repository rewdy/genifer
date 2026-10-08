# generation-workflow Specification

## Purpose

Defines the end-to-end flow of composing a prompt, reviewing it before sending, running generation asynchronously with cancellation, saving the result to disk, and opening it.

## Requirements

### Requirement: Prompt composition and review

The system SHALL let the user compose a freeform prompt and, before sending, present a review step showing the exact prompt to be submitted, from which the user can either submit it or return to editing.

#### Scenario: Submit from review

- **WHEN** the user confirms the prompt at the review step
- **THEN** generation begins with exactly the reviewed prompt text

#### Scenario: Edit from review

- **WHEN** the user chooses to edit at the review step
- **THEN** the user is returned to the editable prompt with its current text preserved, and no generation is sent

#### Scenario: Empty prompt

- **WHEN** the user attempts to submit with an empty prompt
- **THEN** generation does not start and the user is prompted to enter prompt text

### Requirement: Asynchronous generation with cancellation

The system SHALL perform generation without blocking the interface, SHALL show a progress indicator while a generation is in flight, and SHALL allow the user to cancel the in-flight generation, which aborts the underlying request.

#### Scenario: Progress indicator

- **WHEN** a generation is in flight
- **THEN** the interface remains responsive and shows a progress indicator

#### Scenario: Cancel in flight

- **WHEN** the user presses the cancel key during generation
- **THEN** the request is aborted and the user is returned to prompt composition without a saved image

### Requirement: Saving generated images

The system SHALL save a successfully generated image to the configured output directory using a non-colliding, timestamped file name whose extension matches the image's media type, and SHALL report the saved file path to the user.

#### Scenario: Save on success

- **WHEN** a generation succeeds
- **THEN** the image is written to the output directory with an extension matching its media type, and the resolved path is shown

#### Scenario: Output directory does not exist

- **WHEN** the configured output directory does not yet exist at save time
- **THEN** genifer creates it before writing the file

#### Scenario: Write failure

- **WHEN** the image cannot be written (e.g. permissions)
- **THEN** genifer reports a clear error and does not claim success

### Requirement: Reporting generation cost

After a successful generation, when the provider reports an actual cost, the system SHALL display that cost to the user alongside the saved result.

#### Scenario: Cost shown on success

- **WHEN** a generation succeeds and a cost was reported for it
- **THEN** the result display includes the actual cost in US dollars

#### Scenario: Cost unavailable

- **WHEN** a generation succeeds but no cost was reported
- **THEN** the result display omits the cost rather than showing a misleading value

### Requirement: Open and auto-open behavior

After saving, the system SHALL offer to open the image in the system viewer. When an `auto_open` setting is enabled, the system SHALL open the image automatically without prompting. The command used to open files SHALL default per operating system and SHALL be overridable by configuration.

#### Scenario: Offer to open (auto-open off)

- **WHEN** auto-open is disabled and an image has been saved
- **THEN** the user is offered an action to open the image, and dismissing it leaves the file saved

#### Scenario: Auto-open on

- **WHEN** auto-open is enabled and an image has been saved
- **THEN** the image is opened in the system viewer without an extra prompt

#### Scenario: Open command override

- **WHEN** an open command is configured
- **THEN** that command is used to open the file instead of the OS default

### Requirement: Surfacing generation failures

The system SHALL present generation failures as human-readable messages distinguishing at least insufficient credit, generation failure, authentication failure, and cancellation, and SHALL return the user to a state from which they can retry or adjust.

#### Scenario: Credit error shown

- **WHEN** generation fails due to insufficient credit
- **THEN** a message explaining the credit problem is shown rather than a raw error

#### Scenario: Retry after failure

- **WHEN** generation fails for a retryable reason
- **THEN** the user can retry without re-entering the prompt
