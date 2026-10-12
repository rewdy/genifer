# Spec Delta

## Purpose

Defines the actions a user can take on the result screen after a generation
completes, and precisely which composed parameters each action preserves or
clears, so iteration (same prompt / new model, fresh start, or exact re-run) is
direct and predictable.

## ADDED Requirements

### Requirement: Result screen offers distinct post-generation actions on success

After a generation succeeds, the result screen SHALL offer, in addition to
opening the saved image and quitting, three distinct actions: try again with a
new model, start a new image, and re-generate. Each action's key hint SHALL be
shown on the result screen.

#### Scenario: Successful generation shows all post-generation actions
- **WHEN** a generation completes successfully
- **THEN** the result screen presents controls for open, try-again-with-new-model, new-image, re-generate, and quit

#### Scenario: Failed generation does not show success-only actions
- **WHEN** a generation fails
- **THEN** try-again-with-new-model, new-image, and re-generate are not offered, and the existing retry (for retryable failures) and new/quit behavior is unchanged

### Requirement: Try again with a new model preserves composed parameters

The try-again-with-a-new-model action SHALL return the user to the model picker
while preserving the entered prompt text, the selected aspect ratio, and any
attached reference images, so that selecting a model proceeds to a prompt screen
pre-populated with those values.

#### Scenario: Prompt and parameters survive model reselection
- **WHEN** the user chooses try-again-with-a-new-model on the result screen
- **THEN** the model picker is shown, and after selecting a model the prompt text, aspect ratio (where offered by the new model), and reference images from the previous generation are still present

#### Scenario: Reference images dropped when new model cannot accept them
- **WHEN** reference images were attached and the newly selected model does not accept reference images
- **THEN** the reference images are not sent, and the user is informed they were dropped

### Requirement: New image clears composed parameters

The new-image action SHALL return the user to the prompt screen with the prompt
text and reference images cleared, while keeping the currently selected model.
No seed SHALL be carried over from the previous generation; for a seed-capable
model the next generation mints a fresh seed per the seed-control capability.

#### Scenario: New image starts from an empty prompt
- **WHEN** the user chooses new-image on the result screen
- **THEN** the prompt screen is shown with an empty prompt and no reference images attached, and no seed from the previous generation is carried over

### Requirement: Re-generate repeats the completed generation

The re-generate action SHALL immediately start a new generation using the same
model, prompt, aspect ratio, and reference images as the generation that just
completed, without returning the user to the prompt or picker first. For a
seed-capable model, the seed is governed by the seed-control capability
(reuse the same seed to reproduce, or use a new seed to vary); for a model
without seed support, the request repeats as sent.

#### Scenario: Re-generate reuses prompt and parameters
- **WHEN** the user chooses re-generate on the result screen
- **THEN** a new generation begins with the same model, prompt, aspect ratio, and reference images as the just-completed generation

#### Scenario: Re-generate seed follows the seed-control capability
- **WHEN** the user chooses re-generate for a seed-capable model
- **THEN** the seed sent is determined by the user's reuse-or-vary choice as defined by the seed-control capability
