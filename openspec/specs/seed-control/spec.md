# seed-control Specification

## Purpose

For seed-capable models, genifer auto-generates the seed so the user does not
have to, shows the seed in use so the user knows one is active and what it is,
and lets the user reproduce a result or deliberately vary it from the result
screen — keeping the user informed without asking for input they don't need to
give.

## Requirements

### Requirement: Seed is auto-generated for seed-capable models

For a model whose capabilities report seed support, the system SHALL generate a
seed value itself and send it with the generation, without requiring the user to
supply one. For a model that does not support seed, no seed SHALL be generated
or sent.

#### Scenario: Seed minted automatically when supported
- **WHEN** the active model supports a seed and the user starts a generation without specifying one
- **THEN** genifer generates a seed value and includes it in the request

#### Scenario: No seed for unsupported models
- **WHEN** the active model does not support a seed
- **THEN** no seed is generated and none is sent

### Requirement: The seed in use is shown to the user

For a seed-capable model, the system SHALL display the seed value that was used
for a generation so the user knows a seed is in effect and what its value is.
For a model that does not support seed, no seed SHALL be shown.

#### Scenario: Result shows the seed used
- **WHEN** a generation completes for a seed-capable model
- **THEN** the result presentation shows the seed value that was used

#### Scenario: No seed shown for unsupported models
- **WHEN** a generation completes for a model that does not support seed
- **THEN** no seed value is shown

### Requirement: Re-generate lets the user reproduce or vary via the seed

When the user re-generates from the result screen for a seed-capable model, the
system SHALL make clear what happens to the seed and SHALL let the user either
reuse the same seed (reproduce the image) or use a new seed (produce a
variation). The default behavior SHALL favor the guiding principle: communicate
the seed handling rather than silently changing it, without forcing the user to
type a value.

#### Scenario: Reuse the seed to reproduce
- **WHEN** the user chooses to re-generate reusing the current seed
- **THEN** the same seed is sent again so the result reproduces the prior image

#### Scenario: Bump the seed for a variation
- **WHEN** the user chooses to re-generate with a new seed
- **THEN** a newly generated seed is sent, and the new seed is shown as the one now in use

#### Scenario: Seed handling is communicated on re-generate
- **WHEN** the user triggers re-generate for a seed-capable model
- **THEN** the UI indicates whether the seed is being reused or has been changed, rather than changing it silently
