# Tasks

## 1. Project scaffolding

- [x] 1.1 Initialize the Go module (`go.mod`) and create the package layout from design.md (`main.go`, `internal/config`, `internal/provider`, `internal/provider/openrouter`, `internal/gen`, `internal/tui`); verify `go build ./...` succeeds on an empty skeleton
- [x] 1.2 Add pinned dependencies (Bubble Tea, lipgloss, bubbles, a YAML parser) to `go.mod` and verify `go mod tidy` resolves and `go build ./...` still succeeds

## 2. Config, state, and value resolver

- [x] 2.1 Implement the generic lazy value resolver (literal / `{env:NAME}` / `{cmd:...}`, single-level, trimmed command stdout) and verify unit tests cover each form plus missing-env and command-failure errors (config spec: value resolution)
- [x] 2.2 Implement `config.yaml` loading from the platform config dir with defaults-on-missing and a clear error on malformed YAML; verify unit tests for present/missing/malformed cases (config spec: config loading)
- [x] 2.3 Implement `state.json` read/write for last-used model, never writing `config.yaml`, degrading gracefully when absent/invalid; verify unit tests for round-trip, missing, and invalid-state cases (config spec: state persistence)
- [x] 2.4 Document config schema and the `{env:}`/`{cmd:}` directives in `docs/config.md`; verify the documented example `config.yaml` loads without error

## 3. Provider interface

- [x] 3.1 Define the `Provider` interface and provider-agnostic types (`Model`, `Capabilities`, `GenerateRequest`, `ReferenceImage`, `GenerateResult`) in `internal/provider`; verify `go build ./...` and that no provider-specific fields appear in these types (image-provider spec: provider abstraction)

## 4. OpenRouter provider

- [x] 4.1 Implement model discovery against `GET /api/v1/images/models`, mapping response into `[]Model` with capabilities (reference-image support, supported params); verify a unit test maps a captured sample response and distinguishes auth failure from other errors (image-provider spec: model discovery)
- [x] 4.2 Implement `Generate` against `POST /api/v1/images` with Bearer auth, `n=1`, optional `aspect_ratio`/`seed`, decoding `b64_json` and deriving extension from `media_type`; verify a unit test decodes a sample success response (image-provider spec: generation)
- [x] 4.3 Add reference-image (`input_references[]`) support, rejecting reference images for models that do not accept them before sending; verify unit tests for the supported and unsupported-model paths (image-provider spec: reference images)
- [x] 4.4 Map 402/502/auth responses to distinct errors (insufficient credit / generation failed-retryable / authentication) and honor `ctx` cancellation aborting the request; verify unit tests for each mapped error and a cancellation test (image-provider spec: error mapping, cancellable requests)
- [x] 4.5 Document required credentials and attribution headers in `docs/openrouter.md`; verify the documented env-var setup matches what the provider reads

## 5. Generation workflow

- [x] 5.1 Implement prompt composition with a review step (submit vs. edit, reject empty prompt); verify unit tests for submit-exact-text, edit-preserves-text, and empty-prompt-blocked (generation-workflow spec: composition and review)
- [x] 5.2 Implement saving a result to the output dir with a timestamped, collision-free name and media-type extension, creating the dir if missing, erroring clearly on write failure; verify unit tests for success, dir-created, and write-failure (generation-workflow spec: saving)
- [x] 5.3 Implement open / auto-open with per-OS default command and config override (resolved via the value resolver); verify a unit test selects the correct default per OS and honors an override (generation-workflow spec: open behavior)
- [x] 5.4 Wire async generation with cancellation and failure surfacing (credit/failed/auth/cancelled mapped to messages, retry without re-entering prompt); verify a unit/integration test drives success and each failure/cancel path through the workflow (generation-workflow spec: async generation, surfacing failures)

## 6. TUI

- [x] 6.1 Build the full-screen Bubble Tea shell with the styled gradient header, alt-screen enter/exit, and resize reflow; verify manual run shows the header full-screen and restores the terminal on quit (tui spec: application shell)
- [x] 6.2 Build the model picker fed by `Provider.Models`, pre-selecting the last-used model when still available, persisting selection, and showing an actionable error when the list fails; verify manual run and a unit test for pre-selection/fallback logic (tui spec: model selection)
- [x] 6.3 Build the capability-adaptive generate form (hide unsupported params, offer reference-image attach only when supported, update on model switch); verify a unit test derives controls from capabilities for a ref-capable and a ref-incapable model (tui spec: adaptive form)
- [x] 6.4 Add status display (idle/generating/saved/error) and keybinding hints (submit, cancel=ESC, open, quit) wired to the workflow via `tea.Cmd`; verify manual run shows status transitions and that ESC cancels an in-flight generation (tui spec: status and keybindings)

## 7. Integration

- [x] 7.1 Wire `main.go` end to end (load config -> resolve key lazily -> construct OpenRouter provider -> launch TUI) and verify a full manual run: pick model, enter prompt, review, generate, image saved, open offered/auto-opened
- [x] 7.2 Verify `go build ./...` and `go vet ./...` are clean and `go test ./...` passes across all packages
- [x] 7.3 Write `README.md` (install, configure, run, example session) and verify the documented quickstart commands work as written
