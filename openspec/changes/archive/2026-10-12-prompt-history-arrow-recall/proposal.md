# Proposal

## Why

The prompt-history change gave genifer a persisted history and a dedicated
history view (opened with `h` from the picker or `ctrl+p` while composing), but
reaching a past prompt still means leaving the compose screen for the list.
Shells and most REPL-style prompts let you recall recent entries inline with the
Up/Down arrows — press Up to walk back through what you typed before, Down to
come forward again, and Down past the newest to get your in-progress text back.
Adding that to the compose prompt makes quick iteration ("run that last thing
again, tweak one word") feel immediate without a detour through the list.

Prior art is instructive here: Claude Code's interactive prompt binds Up/Down to
history, and its issue tracker is full of complaints from getting the edges
wrong — hijacking the arrows so the cursor can no longer move through a
multi-line prompt, and destroying an unsent draft with no way to get it back.
The readline/shell convention avoids both: recall only at the text edges, and
keep the in-progress line as a restorable slot. This change follows that
convention deliberately.

## What Changes

- On the compose screen, make Up/Down recall prompt history inline, scoped to
  the compose prompt only (never the picker, review, aspect dialog, or history
  view):
  - **Up** recalls the previous (older) entry **only when the cursor is on the
    first line** of the prompt; otherwise Up moves the cursor up as usual.
  - **Down** moves to the next (newer) entry **only when the cursor is on the
    last line**; otherwise Down moves the cursor down as usual.
  - Stepping **Down past the newest** entry restores the in-progress draft that
    was present when recall started (which may be empty), exactly like a shell.
- **Preserve the draft**: the text the user had typed before the first recall is
  snapshotted and always restorable; recall never destroys unsent text.
- **Reset on edit**: any manual edit to the prompt ends the current recall
  session, so the next Up starts again from the newest entry.
- Recall draws from the same most-recent loaded entries as the history view and,
  consistent with selecting from that view, reapplies the entry's aspect ratio
  and seed where the active model supports them.

## Capabilities

### New Capabilities

<!-- None. -->

### Modified Capabilities

- `prompt-history`: adds inline Up/Down history recall on the compose prompt
  (edge-aware navigation, draft preservation, reset-on-edit) as an additional
  way to reuse a past prompt, alongside the existing history view.

## Impact

- `internal/tui/model.go`: `handleComposeKey` gains edge-aware Up/Down handling
  that delegates to recall only at the first/last line, plus a recall-session
  reset when the prompt is edited.
- `internal/tui/history.go` (or nearby): recall-session state on `Model` (the
  saved draft and the current recall index) and the step-older / step-newer /
  restore-draft logic, reusing the loaded recent entries and the existing
  reuse-entry path that applies prompt, aspect ratio, and seed.
- `internal/tui/view.go`: compose footer hint mentioning Up/Down history recall.
- `README.md`: document inline Up/Down recall alongside the history view.
