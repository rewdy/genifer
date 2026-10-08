# Tasks

## 1. Persist last-used aspect ratio

- [x] 1.1 Add `LastAspectRatio string \`json:"last_aspect_ratio,omitempty"\`` to `State` in `internal/config/state.go` and update the struct doc comment. Verify `go build ./...` succeeds.
- [x] 1.2 Add a round-trip test in `internal/config/state_test.go` asserting `LastAspectRatio` survives `SaveState`/`LoadState` and that a zero/missing/invalid state yields an empty value. Verify `go test ./internal/config/` passes.

## 2. Default-selection helper

- [x] 2.1 Add `pickAspectRatio(offered []string, last string) string` in `internal/tui/form.go` mirroring `preselectModel`: return `""` when `offered` is empty, `last` when it is in `offered`, else `offered[0]`. Verify `go build ./...` succeeds.
- [x] 2.2 Add a table-driven test (sibling to `TestPreselectModel`) in `internal/tui/tui_test.go` covering: last present, last absent, empty last, empty offered. Verify `go test ./internal/tui/` passes.

## 3. Selector state and generation wiring

- [x] 3.1 Add an `aspectRatio string` field (current selection) and a transient `aspectCursor int` (dialog highlight) to the compose section of `Model` in `internal/tui/model.go`, plus a `phaseAspect` phase constant. Verify `go build ./...` succeeds.
- [x] 3.2 Compute `m.aspectRatio = pickAspectRatio(currentCaps().AspectRatios, config.LoadState(m.deps.StatePath).LastAspectRatio)` whenever the active model is set/changed (model-pick `enter` branch in `handlePickerKey`). Verify entering compose for a ratio-capable model sets a non-empty selection via a test.
- [x] 3.3 Set `AspectRatio: m.aspectRatio` in `Model.composeDraft()`. Add/extend a test asserting the draft carries the selected ratio. Verify `go test ./internal/tui/` passes.

## 4. Selector dialog and rendering

- [x] 4.1 In `handleComposeKey`, when the current model offers aspect ratios, intercept `ctrl+a` to open the dialog (`phase = phaseAspect`, seed `aspectCursor` to the current selection's index); otherwise fall through to the textarea. Add a `handleAspectKey` handler: `up`/`down` move the cursor (clamped), `enter` sets `aspectRatio = offered[aspectCursor]` and returns to compose, `esc` returns to compose unchanged. Verify handler tests cover move/confirm/cancel and that `ctrl+a` is ignored for a no-ratio model.
- [x] 4.2 Update `composeView` in `internal/tui/view.go` to render the current aspect-ratio value and the open-dialog hint when ratios are offered, and nothing when none are. Add an `aspectView` (dialog) rendering all offered ratios with the cursor row marked, wired into `bodyView` for `phaseAspect`. Verify via view tests (ANSI-stripped `strings.Contains`) that the selected value and hint appear in compose, that the dialog lists every offered ratio, and that neither appears for a no-ratio model.
- [x] 4.3 Add footer key hints: a compose-phase hint for opening the aspect dialog (shown only when ratios are offered), and a `phaseAspect` key set (move/select/cancel). Verify via a `footerView` test.

## 5. Save on generate

- [x] 5.1 On generate, persist the selected aspect ratio to `state.json` using read-modify-write (load current `State`, set `LastAspectRatio`, save) so the remembered model is preserved; likewise ensure the model-pick save preserves `LastAspectRatio`. Verify a test that generates then reads `state.json` shows both fields retained.

## 6. Docs and integration check

- [x] 6.1 Update `docs/config.md` and `README.md` to note the aspect-ratio selection dialog (opened with `ctrl+a`) and that the last-used value is remembered in `state.json`. Verify the documented key matches the implemented binding.
- [x] 6.2 Run `go build ./...`, `go test ./...`, and `go vet ./...` (or `just check`) and confirm all pass.
