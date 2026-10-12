# Tasks

## 1. History storage

- [x] 1.1 Add `internal/config/history.go` with `HistoryCache`/`HistoryEntry`, `HistoryPath()`, versioned best-effort `LoadHistory`, and `SaveHistory` mirroring `pricing.go`; verify a test covers missing, unparsable, and wrong-version files all yielding an empty history.
- [x] 1.2 Add an `Append` helper that prepends the newest entry and retains all entries (no truncation or deletion); verify a test asserts newest-first ordering and that appending never drops previously recorded entries.

## 2. Config opt-out

- [x] 2.1 Add a `history` config field defaulting to enabled, ensuring an absent key means enabled; verify a table-driven config test over absent / `true` / `false` inputs resolves to the correct enabled state.
- [x] 2.2 Document the `history` option and `history-cache.json` in `docs/config.md`; verify the documented key name and default match the config struct.

## 3. Record on success

- [x] 3.1 In the `generatedMsg` handler, write a history entry via a background `tea.Cmd` only when history is enabled and the outcome succeeded, treating write failure as non-fatal; verify a test asserts an entry is written on success and skipped on failure/cancel and when disabled.

## 4. History view

- [x] 4.1 Add a `phaseHistory` and a history list view (reusing the picker's `list` idiom) that loads at most the newest 25 entries, newest-first, with an empty-state and a disabled-state message, and a message pointing to the resolved config-dir history file when the user pages past the loaded entries (and more exist in the store) or filters to no matches; verify a view test asserts newest-first rendering, the 25-entry load limit, the empty and disabled messages, and the full-history pointer on end-of-list-with-more and on empty filter.
- [x] 4.2 Add a key to open the history view and wire `esc` to return and `enter` to select; verify a test drives open → select → return transitions.
- [x] 4.3 On selecting an entry, transition to `phaseCompose` pre-filled with the prompt and apply aspect ratio and seed where the active model supports them; verify a test asserts the prompt is populated and parameters applied only when supported.

## 5. Integration

- [x] 5.1 Update `README.md` to mention history and its opt-out; verify the described key and behavior match the implementation.
- [x] 5.2 Run `go build ./...`, `go test ./...`, and `go vet ./...` (or `just check`) and verify all pass.
