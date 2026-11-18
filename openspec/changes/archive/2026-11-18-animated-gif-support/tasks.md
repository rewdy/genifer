# Tasks

## 1. Provider capability and request guard

- [x] 1.1 Add `SupportsAnimatedGIF` to `provider.Capabilities` and set it in the OpenRouter adapter when `output_format` advertises `gif`; verify capability tests cover a GIF-capable and a non-GIF-capable descriptor and pass.
- [x] 1.2 Guard `openrouter.Generate` so `output_format: "gif"` is omitted for models that do not advertise it and transparent background is omitted when GIF output is requested; verify request-payload tests assert both cases.
- [x] 1.3 Run `go test ./internal/provider/...` and `go vet ./internal/provider/...` and verify both pass.

## 2. Compose option and GIF file output

- [x] 2.1 Add animated GIF state and a `ctrl+g` toggle to the compose form, show it only via `FormControls` for capable models, display its on/off state and key hint, add a `gif` capability hint, and enforce mutual exclusion with transparency including on model switch; verify TUI tests cover show/hide, both toggle directions, and switching models.
- [x] 2.2 Set `OutputFormat = "gif"` in the draft builder only when the toggle is on and the model supports it; verify generation tests assert the resulting provider request and the absence of a transparent background.
- [x] 2.3 Map `image/gif` to `.gif` in the save extension table; verify a save test writes a GIF response with a `.gif` path.
- [x] 2.4 Run `go test ./internal/tui/... ./internal/gen/...` and verify all tests pass.

## 3. Model picker capability filters

- [x] 3.1 Add picker filter state and apply it when rebuilding model items so only models satisfying every active capability filter reach the list; verify unit tests cover single, multiple, and text-composed filters.
- [x] 3.2 Add picker keys to open/toggle filters and clear them in one action, show active filters in the status area, preserve the selected model where possible, and display an empty-result message; verify TUI tests cover each filter, favorites-first ordering, no-match feedback, and clearing.
- [x] 3.3 Run `go test ./internal/tui/...` and `go vet ./internal/tui/...` and verify both pass.

## 4. History captures and restores settings

- [x] 4.1 Add optional background and output-format fields to `config.HistoryEntry`, record them from the completed draft, and keep the cache version unchanged; verify config round-trip tests show new records persist and older records without the fields still load.
- [x] 4.2 Add one shared restore helper used by history selection and inline recall that applies aspect ratio, seed, transparency, and GIF output only where the selected model supports them and resolves GIF/transparency conflicts in favor of GIF; verify TUI tests cover supported, unsupported, and conflicting settings.
- [x] 4.3 Extend the recall snapshot to include in-progress settings so stepping past the newest entry restores the original prompt and settings; verify the recall test asserts both.
- [x] 4.4 Run `go test ./internal/config/... ./internal/tui/...` and verify all tests pass.

## 5. Documentation and integration

- [x] 5.1 Document GIF support, how capability discovery reports it, and the transparency exclusion in `docs/openrouter.md`; verify the documented request fields match the implemented payload.
- [x] 5.2 Run `just check` and `go run . --version` and verify the build, tests, vet, and version output all succeed.
- [x] 5.3 With `OPENROUTER_API_KEY` set, generate one GIF using a GIF-capable model and verify the saved file has a `.gif` extension and plays as an animation; note that this is a paid manual smoke test that can be skipped when credentials are unavailable.
