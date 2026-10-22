# Tasks

## 1. Research & OpenRouter API Integration

- [x] 1.1 Research OpenRouter API documentation for transparent background parameters and model capability discovery. Verify the parameter name and existence of a capability API.
- [x] 1.2 Update `internal/provider/openrouter/openrouter.go` to include the transparency parameter in image generation API requests. Verify by making a successful API call with the parameter.
- [x] 1.3 Implement model capability discovery within `internal/provider/openrouter/` (or a configuration fallback if API not available) to identify models supporting transparency. Verify that the capability is correctly identified for a test model.
- [x] 1.4 Cache model transparency capabilities to reduce API calls. Verify caching mechanism is functional and reduces redundant lookups.

## 2. UI Development (`internal/tui/`)

- [x] 2.1 Add a UI element (e.g., checkbox or toggle) to `internal/tui/model.go` for enabling/disabling transparent backgrounds in the image generation form. Verify the UI element appears correctly.
- [x] 2.2 Implement logic in `internal/tui/model.go` to dynamically enable/disable the transparency UI element based on the selected model's transparency capability. Verify the UI element's state changes correctly with model selection.
- [x] 2.3 Add a user-facing message in `internal/tui/model.go` to indicate if the selected model does not support transparent backgrounds. Verify the message is displayed when applicable.

## 3. Generation Workflow (`internal/gen/`)

- [x] 3.1 Integrate the transparency setting from the TUI into the image generation request within `internal/gen/`. Verify that the transparency flag is passed down to the provider.
- [x] 3.2 Handle cases where transparency is requested but not supported by the model, ensuring the image is generated without transparency and a warning is logged/displayed. Verify warning is issued and image is non-transparent.

## 4. Testing & Verification

- [x] 4.1 Write unit tests for the OpenRouter API integration (`internal/provider/openrouter/`). Verify that requests are correctly formed with/without the transparency parameter.
- [x] 4.2 Write integration tests covering the UI, generation workflow, and OpenRouter integration. Verify that enabling transparency in the UI results in a transparent image (if supported) and appropriate feedback for unsupported models.
- [ ] 4.3 Manually test the transparent image generation feature with various OpenRouter models, including those expected to support transparency and those not. Verify expected behavior and UI feedback.
