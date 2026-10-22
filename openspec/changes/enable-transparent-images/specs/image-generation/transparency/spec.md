# Spec Delta

## Purpose

To enable the generation of images with transparent backgrounds when supported by the underlying image generation model.

## ADDED Requirements

### Requirement: User can request transparent background
The system SHALL provide an option for the user to request a transparent background for generated images.

#### Scenario: Transparency option is presented
- **WHEN** the user interacts with the image generation UI
- **THEN** an option to enable/disable transparent background is visible and selectable

#### Scenario: Transparency request is honored
- **WHEN** the user selects the transparent background option and initiates image generation
- **THEN** the system attempts to generate an image with a transparent background, if the selected model supports it

### Requirement: System indicates transparency support
The system SHALL indicate whether the currently selected image generation model supports transparent backgrounds.

#### Scenario: Model supports transparency
- **WHEN** a model that supports transparent backgrounds is selected
- **THEN** the UI clearly indicates that transparent background generation is available

#### Scenario: Model does not support transparency
- **WHEN** a model that does not support transparent backgrounds is selected
- **THEN** the UI indicates that transparent background generation is not available and the option is disabled or hidden
