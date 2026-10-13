# Tasks

## 1. Recall session state and stepping

- [x] 1.1 Add recall-session state to `Model` (`recallIndex int` defaulting to the not-recalling sentinel, and `recallDraft string`), and helpers to step older, step newer, and restore the draft over `m.historyEntries`, reusing `reuseHistoryEntry` to apply a recalled entry (prompt + aspect ratio + seed where supported) and setting only the prompt (clearing `reuseSeed`) when restoring the draft; verify tests for step-older from the draft (snapshots the draft, applies newest), stepping to older entries (clamped at the last loaded), and stepping newer past the newest (restores the exact draft, including empty).

## 2. Edge-aware Up/Down in compose

- [x] 2.1 In `handleComposeKey`, handle `up` as recall-previous only when the cursor is on the first line (else forward to the textarea) and `down` as recall-next only when the cursor is on the last line (else forward), with both a no-op fall-through when history is disabled or no entries are loaded; verify tests assert recall at the edges and plain cursor movement off the edges for a multi-line prompt, and that both are inert with no loaded history.
- [x] 2.2 End the recall session (reset to the not-recalling sentinel and drop the saved draft) when an editing key reaches the textarea, so the next `up` at the first line starts a fresh session from the newest entry; verify a test recalls an entry, edits, then asserts `up` recalls the newest again.

## 3. Scope and hints

- [x] 3.1 Confirm recall is reachable only in `phaseCompose` (no Up/Down recall in picker, review, aspect dialog, or history view) and add a compose footer hint for Up/Down history recall; verify a footer test asserts the hint on compose and a test asserts Up/Down do not recall outside compose.

## 4. Documentation and integration

- [x] 4.1 Update `README.md` to document inline Up/Down history recall on the compose screen alongside the history view; verify the described behavior and keys match the implementation.
- [x] 4.2 Run `go build ./...`, `go test ./...`, and `go vet ./...` (or `just check`) and verify all pass.
