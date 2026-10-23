# Tasks

## 1. Persist favorites in app-owned state

- [x] 1.1 Add `Favorites []string` (`json:"favorites,omitempty"`) to `config.State` in `internal/config/state.go` and verify `go build ./...` passes
- [x] 1.2 Add a set-conversion helper for favorites and verify with a table-driven test covering empty, duplicate, and normal inputs
- [x] 1.3 Extend `internal/config/state_test.go` to verify favorites round-trip through `SaveState`/`LoadState`, that saving favorites preserves `LastModel`/`LastAspectRatio`, and that a missing/invalid file yields an empty set; run `go test ./internal/config`

## 2. Order the picker with favorites first

- [x] 2.1 Add a `favoritesFirst(models []provider.Model, favs map[string]struct{}) []provider.Model` helper in `internal/tui` that stable-partitions favorites in provider order, and verify with table-driven tests (none, all, mixed, stale IDs ignored)
- [x] 2.2 Replace index-based list selection with a select-by-model-ID helper in `internal/tui/model.go`, used by `rebuildItems` and the `modelsLoadedMsg` pre-selection path, and verify existing picker/selection tests still pass
- [x] 2.3 Add `favorites map[string]struct{}` to `Model`, load it when models load, and build `modelItem` values with a `favorite bool` in `rebuildItems`; verify `go test ./internal/tui` covers that a favorited model sorts first and the list still reports the same model count

## 3. Toggle favorites and render the indicator

- [x] 3.1 Add the `f` keybinding in `handlePickerKey` that toggles the highlighted model's favorite state, writes `state.json`, rebuilds items, and keeps the highlight on the toggled model; verify with a Bubble Tea model test asserting the favorite is persisted and the highlight follows the model
- [x] 3.2 Verify the toggle key is ignored while the picker filter input is active (typed into the query, no state change) with a test
- [x] 3.3 Add the `favoriteStar` yellow to `internal/tui/styles.go`, render a yellow `★` prefix in `modelDelegate` (blank padding on non-favorited rows so names stay aligned) without disturbing name/detail/padding, and add the `f` favorite hint to the picker help row in `internal/tui/view.go`; verify with a render test asserting the yellow star appears only on favorited rows, alignment is unchanged, and the picker hint is present

## 4. Integration checks

- [x] 4.1 Verify stale favorite IDs produce no extra rows and no error by loading a state file with an unknown model id into a test model, then confirm the picker renders only provider models
- [x] 4.2 Verify `config.yaml` is untouched after toggling favorites by asserting the file bytes are unchanged in a test
- [x] 4.3 Run `just check` (build + test + vet) and confirm everything passes
