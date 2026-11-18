# Spec Delta

## Purpose

Lets users generate animated GIF images with the models that advertise GIF output, from capability discovery through the compose option, guarded request, and correctly named saved file.

## ADDED Requirements

### Requirement: Animated GIF support is discovered and reported

The system SHALL determine animated GIF support from provider-advertised model output formats (for example an `output_format` enum containing `gif`) and expose it as a provider-agnostic capability. A model that does not advertise GIF output SHALL be reported as not supporting animated GIF. Support SHALL NOT be inferred from model names or a hand-maintained list of model identifiers, so newly offered animation-capable models become available without a code change.

#### Scenario: Model advertises GIF output

- **WHEN** model discovery reports `gif` among a model's supported output formats
- **THEN** that model is reported as supporting animated GIF generation

#### Scenario: Model does not advertise GIF output

- **WHEN** model discovery does not report `gif` among a model's supported output formats
- **THEN** that model is reported as not supporting animated GIF generation

### Requirement: Compose offers an animated GIF option for capable models

The generate form SHALL present an animated GIF toggle only when the selected model supports animated GIF output. The toggle SHALL show its current on/off state and SHALL be discoverable through a key hint. Animated GIF output and transparent backgrounds are mutually exclusive: turning one on SHALL turn the other off. Switching to a model that does not support animated GIF SHALL hide the toggle and clear the selection, and generation proceeds with the model's default output.

#### Scenario: Capable model shows the option

- **WHEN** the selected model supports animated GIF
- **THEN** the compose form shows the animated GIF toggle with its current state and a key hint

#### Scenario: Incapable model hides the option

- **WHEN** the selected model does not support animated GIF
- **THEN** the compose form does not present the animated GIF toggle

#### Scenario: Enabling GIF clears transparency

- **WHEN** transparent background is on and the user turns animated GIF on
- **THEN** transparent background turns off before the generation request is built

#### Scenario: Enabling transparency clears GIF

- **WHEN** animated GIF is on and the user turns transparent background on
- **THEN** animated GIF output turns off before the generation request is built

#### Scenario: Switching to an incapable model

- **WHEN** animated GIF is on and the user selects a model that does not support it
- **THEN** the form hides the toggle and the next generation uses the model's default output format

### Requirement: GIF requests are guarded and saved as GIF

The system SHALL request GIF output only for models that report animated GIF support; for all other models it SHALL omit the GIF output request and use the provider default. A generated GIF SHALL be saved with a `.gif` file extension derived from the response media type, and all other outputs SHALL keep the extension matching their media type.

#### Scenario: Capable model requests GIF

- **WHEN** the user enables animated GIF for a model that supports it and generates
- **THEN** the generation request asks for GIF output and the result is saved with a `.gif` extension

#### Scenario: Incapable model never requests GIF

- **WHEN** a GIF output request would be sent for a model that does not support it
- **THEN** the system omits the GIF output request before calling the provider and uses the default output format

#### Scenario: Saved extension follows the response

- **WHEN** the provider returns an image whose reported media type is not `image/gif`
- **THEN** the file is saved with the extension matching that reported media type rather than `.gif`
