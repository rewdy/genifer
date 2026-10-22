# Proposal

## Why

This change addresses the need to enable generating transparent background images, which is a common requirement for image manipulation and integration into various visual contexts. The OpenRouter API is believed to support this functionality via a parameter, and enabling this feature will enhance the utility of the image generation tool.

## What Changes

- Introduce a new option in the UI to enable/disable transparent backgrounds for generated images.
- Consume the OpenRouter API to pass the transparency parameter to supported models.

## Capabilities

### New Capabilities
- `image-generation/transparency`: Provides the ability to generate images with transparent backgrounds.

### Modified Capabilities
- None

## Impact

- `internal/provider/openrouter/`: Modifications to integrate the transparency parameter with OpenRouter API calls.
- `internal/tui/`: UI changes to expose the transparency option to the user.
- `internal/gen/`: Potential changes to the image generation workflow to handle transparent images.
