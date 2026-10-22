# Tasks

## 1. Config settings

- [x] 1.1 Add `SmartFilenames bool` (`yaml:"smart_filenames"`) and `FilenameModel Value` (`yaml:"filename_model"`) to `config.Config` in `internal/config/config.go`; verify `go build ./...` passes and a `config_test.go` case loads YAML with both keys set/unset into the expected struct values
- [x] 1.2 Document both keys with defaults in `starterTemplate` in `internal/config/starter.go`; verify the existing starter test (which loads the generated YAML) still passes via `go test ./internal/config/`
- [x] 1.3 Add `SmartFilenames bool` and `FilenameModel string` to `config.EditableSettings` and write them as top-level keys in `WriteSettings` (`setScalarKey` lines) in `internal/config/writer.go`; verify a `writer_test.go` case asserts both keys are written/replaced and other settings are preserved

## 2. Provider text completion

- [x] 2.1 Add `TextRequest{Model, Prompt string}` and the `GenerateText(ctx, TextRequest) (string, error)` method to the `Provider` interface in `internal/provider/provider.go`; verify `go build ./...` fails only where the method is not yet implemented
- [x] 2.2 Implement `GenerateText` in `internal/provider/openrouter/openrouter.go` — POST to `/chat/completions` with a short system+user `messages` payload, reuse `authHeader`, `c.http`, and `mapTransportError`/`mapHTTPError`, decode `choices[0].message.content`, trim it, and honor ctx cancellation; verify an `openrouter` test using a stub HTTP server asserts the returned text, auth header, and that a 401/402 map to `ErrAuth`/`ErrInsufficientCredit`

## 3. Smart filename generation in gen

- [x] 3.1 Change `gen.Save` to `Save(dir string, res provider.GenerateResult, base string)` where `base == ""` keeps the timestamp; update all existing callers to pass `""`; verify `go test ./internal/gen/` passes with existing save tests unchanged in behavior
- [x] 3.2 Add `sanitizeBaseName(string) string` to `internal/gen` (filesystem-safe chars only, no separator/extension, length-bounded, empty-on-junk); verify table-driven tests cover spaces, slashes, unicode, extension stripping, overlong input, and empty result
- [x] 3.3 Add a `NameOptions{Enabled bool; Model string}` param to `gen.Run` and a `nameFor(ctx, p, d, opts)` helper that returns a sanitized base name or `""` on disabled/no-model/error/empty/cancel; call it between `Generate` and `Save`; verify tests with a fake provider cover: enabled+good name used, disabled → timestamp, model-empty → timestamp, provider error → timestamp, and that the image is still saved when naming fails/cancels

## 4. TUI config editor integration

- [x] 4.1 Add `fieldSmartFilenames` and `fieldFilenameModel` to the `configField` enum and corresponding `bool`/`textinput.Model` to `configState`, seeded in `newConfigState`/`openConfig` from `m.deps.Config` in `internal/tui/config.go`; verify `go build ./...` passes
- [x] 4.2 Handle `space` to toggle smart filenames and route text input to the filename-model field in `handleConfigKey`; render both rows plus a faint `https://openrouter.ai/models` helper line in `configView`; verify the config view renders the toggle, the model field, and the link (manual run of `go run .` config view or a view-string test)
- [x] 4.3 Persist both values via `WriteSettings` and apply them live to `m.deps.Config` in `confirmConfig`; update the config-phase footer help in `internal/tui/view.go` if needed; verify a test (or manual check) confirms the written `config.yaml` round-trips and the running session uses the new values on the next generation

## 5. Wire TUI generation to naming + docs

- [x] 5.1 Build `gen.NameOptions` from `m.deps.Config` and pass it through `runDraft` → `gen.Run` in `internal/tui/model.go`; verify `go build ./...` passes and a generation with smart filenames enabled saves under a derived name while a disabled one keeps the timestamp (manual run with `OPENROUTER_API_KEY`)
- [x] 5.2 Document the `smart_filenames` and `filename_model` settings in `docs/config.md` (and the OpenRouter naming note in `docs/openrouter.md` if relevant); verify the documented keys match the struct and starter template

## 6. Integration verification

- [x] 6.1 Run `go build ./...`, `go test ./...`, and `go vet ./...` (or `just check`) and confirm all pass with no new warnings
- [ ] 6.2 End-to-end check with `OPENROUTER_API_KEY` set: enable smart filenames and a model in the editor, generate, and confirm the saved file has a sanitized descriptive name; then force a naming failure (bad model id) and confirm the save falls back to the timestamped name without a generation error
