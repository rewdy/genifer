# Design

## Context

The `genifer` application currently generates images using various models exposed through the `internal/provider/openrouter/` implementation. The image generation flow involves user input from the TUI (`internal/tui/`), which is then processed by the generation logic (`internal/gen/`) and sent to the selected provider. This design outlines how to extend this flow to enable transparent background image generation. See `proposal.md` for the motivation behind this change.

## Goals / Non-Goals

**Goals:**
- Provide a user interface option in the TUI to request transparent backgrounds for generated images.
- Dynamically determine and display whether the currently selected image generation model supports transparent backgrounds.
- Integrate with the OpenRouter API to pass the transparency parameter when requested and supported.

**Non-Goals:**
- Implementing transparent background generation for image providers other than OpenRouter.
- Developing image post-processing capabilities to remove backgrounds if the selected model does not natively support transparency.
- Modifying the core image generation pipeline to handle new image formats beyond what OpenRouter returns.

## Decisions

1.  **UI Integration:** A checkbox or toggle will be added to the image generation form within `internal/tui/` to allow users to enable or disable transparent backgrounds. This option will be dynamically enabled/disabled based on model support.

2.  **OpenRouter API Parameter:** The OpenRouter API will be inspected to identify the specific parameter (e.g., `transparent_background`, `alpha_channel`, etc.) that controls background transparency. This parameter will be passed as part of the model's request payload within `internal/provider/openrouter/`.

3.  **Model Capability Discovery:**
    *   **Approach:** The application will attempt to dynamically query OpenRouter's model details or capabilities API (if available) to ascertain which models support transparent backgrounds. This information will be cached to avoid repeated API calls.
    *   **Fallback:** If a direct API for capability discovery is not available, a configuration-driven approach will be used where transparency support for specific models is defined in `internal/config/` or within the `openrouter` provider code. This fallback will be noted for future enhancement if OpenRouter introduces a capabilities API.
    *   **UI Reflection:** The TUI will consume this capability information to enable or disable the transparent background option, providing clear feedback to the user.

4.  **Handling Unsupported Requests:** If a user requests a transparent background with a model that does not support it (e.g., due to a race condition or stale cache), the image will be generated without transparency, and a warning message will be displayed to the user. The UI's dynamic disabling of the option should largely prevent this scenario.

## Risks / Trade-offs

-   **Risk:** The OpenRouter API parameter for transparency might not be consistently named or behave identically across all models, or might not exist.
    *   **Mitigation:** Thorough research of OpenRouter API documentation is required. The `internal/provider/openrouter/` code should be designed with flexibility to adapt to variations. If no such parameter exists, this feature will be reconsidered or deferred.
-   **Risk:** Dynamic model capability discovery could introduce latency during model selection or add complexity to the `openrouter` provider.
    *   **Trade-off:** The benefit of accurate, up-to-date model capability information for user experience outweighs the minor complexity increase. Caching will mitigate latency concerns. If latency becomes an issue, an asynchronous lookup or an initial startup pre-fetch could be considered.

## Open Questions

-   What is the precise OpenRouter API parameter name for requesting a transparent background?
-   Does OpenRouter provide a dedicated API endpoint or metadata within existing model endpoints to query specific model capabilities, such as transparency support?
-   Are there any specific image formats (e.g., PNG with alpha channel) that are exclusively used for transparent images by OpenRouter, and does the application need to handle these differently?
