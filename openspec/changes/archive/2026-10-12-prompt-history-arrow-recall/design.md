# Design

## Context

See proposal.md — Why. Compose is a Bubble Tea `textarea.Model` on `Model`
(`internal/tui/model.go`); `handleComposeKey` intercepts specific keys and
otherwise forwards the message to `m.prompt.Update`. The prompt-history change
already added: a loaded set of recent entries (`m.historyEntries`, newest-first,
capped at `historyLoadLimit`), a `reuseHistoryEntry(e)` helper that sets the
prompt and applies aspect ratio / seed where the model supports them, and a
`reuseSeed` field consumed by `composeDraft`. This change adds an inline recall
affordance over that same data and reuse path.

The textarea exposes the current line index and line count (`Line()`,
`LineCount()`), which is enough for edge detection on logical lines.

## Goals / Non-Goals

**Goals:**
- Edge-aware Up/Down recall on the compose prompt that never blocks normal
  cursor movement in a multi-line prompt.
- Shell-style draft preservation: the pre-recall text is always restorable by
  stepping forward past the newest entry.
- Reset the recall session on manual edits.
- Reuse the existing loaded entries and the `reuseHistoryEntry` apply path.

**Non-Goals:**
- Prefix search (recall only entries matching typed text). Shells offer it as an
  opt-in; out of scope here.
- Visual-row edge detection across soft-wrapped lines. Edge detection is on
  logical lines (see Risks). The common single-line and short multi-line cases
  are correct; soft-wrap refinement can come later if needed.
- A new persisted field or storage change — this reads the existing history
  store only.
- Changing the history view or its key bindings.

## Decisions

- **Edge-aware dispatch in `handleComposeKey`.** On `up`, recall only when
  `m.prompt.Line() == 0`; otherwise fall through to the textarea. On `down`,
  recall only when `m.prompt.Line() == m.prompt.LineCount()-1`; otherwise fall
  through. This mirrors readline and avoids Claude Code's regression where the
  arrows always hijacked history and the cursor could no longer traverse a
  multi-line prompt. Alternative — bind a separate key (e.g. `ctrl+p`/`ctrl+n`)
  — rejected: the request is specifically for the familiar Up/Down feel, and
  edge-awareness keeps multi-line editing intact.

- **Recall-session state on `Model`.** Add `recallIndex int` (the position in
  `historyEntries`, with -1 meaning "not recalling / on the draft") and
  `recallDraft string` (the snapshot of the in-progress prompt). Up from the
  draft (`recallIndex == -1`) snapshots the current prompt into `recallDraft`
  and moves to index 0 (newest); further Ups increment toward older entries,
  clamping at the last loaded entry. Down decrements toward newer; stepping past
  index 0 restores `recallDraft` and sets `recallIndex = -1`. Rationale: a tiny
  index + snapshot is the whole shell contract; no new persistence needed.

- **Reuse `reuseHistoryEntry` for applying a recalled entry.** Recall sets the
  prompt and applies aspect ratio / seed through the existing helper so inline
  recall and list selection behave identically. Restoring the draft sets only
  the prompt text (and clears `reuseSeed`) — a draft is the user's own
  in-progress text, carrying no recalled seed.

- **Reset the session on manual edits.** When `handleComposeKey` forwards a
  text-changing key to the textarea while `recallIndex != -1`, set
  `recallIndex = -1` (and drop the saved draft). The next Up then starts a fresh
  session from the newest entry. Detect "editing" as any key that reaches the
  textarea other than pure cursor movement / the recall keys themselves; keep
  the rule simple and conservative (treat the recognized navigation keys as
  non-editing, everything else as editing).

- **Scope strictly to `phaseCompose`.** Recall lives only in `handleComposeKey`.
  Other phases keep Up/Down for their lists/dialogs. This directly avoids the
  class of Claude Code bugs where Up triggered history (or a submit) inside a
  picker or selection UI.

- **No-op when there is nothing to recall.** If history is disabled or no
  entries are loaded, Up/Down at the edges simply fall through to the textarea.

## Risks / Trade-offs

- [Soft-wrapped lines vs. logical lines] → Edge detection uses the textarea's
  logical line index, not visual rows. For a single logical line that wraps
  across several terminal rows, Up at the top immediately recalls rather than
  walking the wrapped rows first. Accepted: it never *blocks* editing (the
  cursor still moves on non-edge lines), and the common cases are correct. A
  visual-row refinement is possible later but is out of scope.
- [What counts as an "edit" that resets recall] → A conservative classifier
  (navigation keys don't reset; everything else does) could occasionally reset
  on a key that didn't change text. Harmless: the worst case is the next Up
  starting from the newest entry, which is the expected shell behavior anyway.
- [Entry has a seed; recall applies it] → Consistent with list selection by
  design, but it means Up-recalling silently sets a seed for a seed-capable
  model. This matches the "reuse a past prompt" intent and the seed is shown on
  the result screen; restoring the draft clears it.

## Open Questions

- None blocking. The exact key classifier for "edit resets recall" is an
  implementation detail chosen during apply.
