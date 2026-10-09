# Spec Delta

## Purpose

Defines a local image provider that generates images through a running
A1111-compatible WebUI over HTTP, so users can generate with models on their own
machine (A1111, Forge, or ComfyUI exposing the compatible API) instead of a
hosted service.

## ADDED Requirements

### Requirement: Local provider conforms to the provider abstraction

The local provider SHALL implement the same provider interface as any other
provider — model discovery, pricing, and generation — so that code above the
provider interacts with it identically to a hosted provider and depends on no
local-provider-specific request or response format.

#### Scenario: Served through the provider interface

- **WHEN** a configured provider instance is of the local type
- **THEN** its models and generations are served through the shared provider interface, indistinguishably from a hosted provider to callers above the interface

### Requirement: Local model discovery

The local provider SHALL discover available models by querying the configured
WebUI's model-listing endpoint, returning each model with a stable identifier
and a human-readable name. Discovery SHALL honor cancellation.

#### Scenario: List installed models

- **WHEN** the local provider's model listing is requested and the WebUI is reachable
- **THEN** it returns the WebUI's installed models, each with an identifier and a name

#### Scenario: Discovery cancelled

- **WHEN** a model-discovery request is cancelled before completion
- **THEN** the provider stops waiting and returns a cancellation outcome rather than a result

### Requirement: Local image generation via the A1111-compatible endpoint

The local provider SHALL generate a single image by posting a prompt to the
WebUI's text-to-image endpoint, translating the request's selected aspect ratio
into output pixel dimensions. It SHALL decode the returned base64 image data and
report a media type usable to choose a file extension. Generation SHALL honor
cancellation.

#### Scenario: Successful local generation

- **WHEN** a generation request with a prompt is submitted and the WebUI returns an image
- **THEN** the provider returns the decoded image bytes with a media-type indication

#### Scenario: Aspect ratio becomes dimensions

- **WHEN** the request carries one of the provider's offered aspect ratios
- **THEN** the provider sends the corresponding output width and height to the WebUI

#### Scenario: Generation cancelled

- **WHEN** a generation request is cancelled before the WebUI responds
- **THEN** the provider stops waiting and returns a cancellation outcome rather than a result

### Requirement: Local models are free

The local provider SHALL report every model's price as free, because generation
runs on the user's own hardware at no per-image service cost. Pricing SHALL be
non-fatal to callers in the same way as any provider's pricing.

#### Scenario: Local model priced free

- **WHEN** pricing is requested for any local model
- **THEN** the provider reports the model as free

### Requirement: Hardcoded local capabilities

Because a local model file does not advertise its own capabilities, the local
provider SHALL report a fixed capability description for its models: seed
supported, a fixed set of offered aspect ratios, and reference images not
accepted. A generation request that includes reference images SHALL be prevented
with a clear error before any call is made.

#### Scenario: Fixed capabilities reported

- **WHEN** the capability description for a local model is requested
- **THEN** it reports seed support and a fixed set of aspect ratios, and reports that reference images are not accepted

#### Scenario: Reference images rejected

- **WHEN** a generation request for a local model includes one or more reference images
- **THEN** generation is prevented with a clear error before any request is sent to the WebUI

### Requirement: Unreachable local endpoint is a graceful failure

When the configured WebUI cannot be reached (for example, it is not running),
the local provider SHALL surface a distinct, actionable failure for discovery
and generation rather than a generic error, so the rest of the application can
treat the provider as offline without aborting.

#### Scenario: WebUI not running during discovery

- **WHEN** model discovery is attempted and the WebUI endpoint cannot be reached
- **THEN** the provider returns an error identifying the endpoint as unreachable, distinct from an authentication failure

#### Scenario: WebUI not running during generation

- **WHEN** a generation is attempted and the WebUI endpoint cannot be reached
- **THEN** the provider returns an unreachable-endpoint error rather than a result
