# Spec Delta

## Purpose

Opt-in generation of a short, meaningful image filename derived from the user's
prompt via a low-cost provider text model, with sanitization, collision
handling, and a guaranteed fallback to the existing timestamped name.

## ADDED Requirements

### Requirement: Opt-in smart filename generation

The system SHALL provide an opt-in "smart filenames" feature controlled by a
user setting that is disabled by default. When disabled, saved images SHALL keep
the existing timestamped naming behavior unchanged. When enabled, the system
SHALL, before saving a successfully generated image, derive the file's base name
by asking a configured low-cost provider text model to produce a short
descriptive name from the generation prompt.

#### Scenario: Disabled by default

- **WHEN** the smart-filenames setting is unset or disabled
- **THEN** generated images are saved with the existing timestamped name and no text model is called

#### Scenario: Enabled produces a descriptive name

- **WHEN** the smart-filenames setting is enabled, a filename model is configured, and a generation succeeds
- **THEN** the saved image's base name is derived from the prompt via the text model, with the extension still matching the image's media type

### Requirement: Smart naming is best-effort and never blocks generation

Smart filename generation SHALL be best-effort: it MUST NOT fail, delay
indefinitely, or discard a successfully generated image. Any failure of the
naming step — the feature enabled but no model configured, a provider or network
error, an empty or unusable model response, or cancellation of the naming
request — SHALL cause the system to fall back to the existing timestamped name
and still save the image. The naming step SHALL honor cancellation.

#### Scenario: Naming request fails

- **WHEN** smart filenames is enabled and the text model request fails or returns an unusable result
- **THEN** the image is still saved using the timestamped name and the user is not shown a generation failure

#### Scenario: Enabled but no model configured

- **WHEN** smart filenames is enabled but no filename model is configured
- **THEN** the image is saved using the timestamped name without attempting a text-model call

#### Scenario: Naming does not block the saved image

- **WHEN** the naming step is slow or cancelled while the image has already been generated
- **THEN** the generated image is still saved rather than lost

### Requirement: Smart filenames are sanitized and collision-free

A model-provided name SHALL be sanitized into a safe base filename before use:
it SHALL contain only filesystem-safe characters, SHALL be bounded in length,
and SHALL NOT include a path separator or file extension. The system SHALL apply
the same non-colliding behavior as timestamped names, so saving multiple images
with the same derived name does not overwrite an existing file. If sanitization
yields an empty name, the system SHALL fall back to the timestamped name.

#### Scenario: Unsafe characters removed

- **WHEN** the model returns a name containing spaces, slashes, or other unsafe characters
- **THEN** the saved base name is sanitized to filesystem-safe characters with no path separator or extension

#### Scenario: Collision avoided

- **WHEN** a derived base name matches a file already present in the output directory
- **THEN** the system saves under a non-colliding variant rather than overwriting the existing file

#### Scenario: Empty after sanitization

- **WHEN** sanitizing the model's response yields an empty string
- **THEN** the system falls back to the timestamped name
