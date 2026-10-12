# Design

## Context

See proposal.md — Why. `internal/config/pricing.go` already establishes the
app-owned-cache pattern this change copies: a versioned struct
(`PricingCache{Version, FetchedAt, Prices}`), a `*Path()` helper under
`config.Dir()`, a `Load*` that returns an empty value on any read/parse/version
error, and a `Save*` that writes `0600` and creates the dir. `state.json`
(`State{LastModel, LastAspectRatio}`) is the other app-owned file; `config.yaml`
is user-owned and never written except by onboarding. A successful generation's
data (prompt via `Draft`, saved path and seed via `Outcome`, model, aspect
ratio) is already available at the `generatedMsg` handler in
`internal/tui/model.go`.

## Goals / Non-Goals

**Goals:**
- A versioned, best-effort `history-cache.json` modeled on `pricing.go` that
  retains every recorded entry.
- A minimal history list view reusing the existing Bubbles `list` idiom already
  used for the model picker, loading only the most-recent entries for
  responsiveness and pointing users to the on-disk file for the long tail.
- A single `HistoryEnabled` config field, default true, that gates all writes.

**Non-Goals:**
- Editing or deleting individual history entries from the UI.
- Thumbnails or image previews in the history list.
- Syncing or exporting history. Searching/filtering beyond what the list widget
  offers for free.

## Decisions

- **New file `internal/config/history.go` mirroring `pricing.go`.** Struct
  `HistoryCache{Version int, Entries []HistoryEntry}` where `HistoryEntry` holds
  `Prompt, Model, AspectRatio string`, `Seed *int64`, `Path string`,
  `CreatedAt time.Time`. `HistoryPath()` → `~/.config/genifer/history-cache.json`.
  `LoadHistory` returns empty on missing/unparsable/wrong-version; `SaveHistory`
  writes `0600`. Rationale: reuse the proven, non-fatal cache contract rather
  than invent a new persistence story.
- **Bound via an `Append` helper that prepends newest and retains all.**
  `Append(cache, entry)` prepends the newest entry; no entries are ever
  truncated or dropped. Keeping ordering newest-first in storage makes the list
  view a direct render. The file is small (prompt text plus metadata), so
  unbounded growth is acceptable; clearing is a manual delete of the file.
  Alternative — truncate to a fixed cap — rejected: it silently loses history,
  which the proposal explicitly avoids.
- **Write on success in the `generatedMsg` handler**, only when
  `Config.HistoryEnabled` and `Outcome.Failure == FailureNone`. The write runs
  in a `tea.Cmd` off the UI loop (like `savePricingCmd`) so a slow disk never
  blocks the render. A write failure is non-fatal (status message at most),
  consistent with "pricing/cache failures are decorative".
- **History view as a new `phaseHistory` using the existing `list.Model`
  idiom, loading only the most-recent entries.** A key from the picker and/or
  compose opens it; the view loads at most a fixed number of the newest entries
  (a view-side constant, e.g. 25) into the list rather than the whole store, so
  rendering stays cheap regardless of how large the file grows. `enter` selects
  an entry and transitions to `phaseCompose` pre-filled; `esc` returns. Reusing
  the picker's list styling keeps the UI consistent and the code small.
- **Point to the on-disk file instead of paging the long tail.** When the user
  pages to the end of the loaded entries and the store holds more than were
  loaded, or when a filter matches none of the loaded entries, the view shows a
  message naming the history file so the user can open it directly. The message
  renders the resolved config-dir path (honoring `XDG_CONFIG_HOME`), falling
  back to the `~/.config/genifer/history-cache.json` form. This avoids building
  load-older/paging UI for a case that is rarely needed — recall is almost
  always of a recent prompt.
- **Config field `HistoryEnabled bool` defaulting true.** Because YAML
  zero-value for bool is `false`, represent it so an unset field means enabled —
  e.g. a `*bool` or an inverted `history_disabled`/explicit default in
  `Default()`. Decision: add `history` as a field whose `Default()` sets it true
  and whose absence in the file keeps the default (the loader already starts
  from `Default()` before unmarshal). Verify unmarshal does not reset it to
  false when the key is absent; if it does, use a pointer and resolve in
  `Default()`/accessor.

## Risks / Trade-offs

- [bool default pitfall] → An absent `history` key must mean enabled, not
  disabled. Mitigation: cover with a config test for absent / `true` / `false`.
- [prompt may contain sensitive text] → History persists prompts in plaintext in
  the config dir (same trust level as the API key already stored there). The
  opt-out is the mitigation; documented in `docs/config.md`.
- [unbounded growth] → Records are retained forever, but each is small (prompt
  text plus metadata), so the file stays modest in practice. The UI never loads
  the whole store (fixed view limit), so view performance is independent of file
  size. A user who wants to reclaim space or clear history deletes the file.
- [large store slows the list] → Only the newest N entries are loaded into the
  list, so rendering cost is bounded regardless of total history size. The long
  tail is reached by opening the file, not by scrolling the TUI.
- [stale saved path] → A recorded `Path` may no longer exist on disk; the view
  reuses the prompt/params regardless and does not require the file to exist.

## Open Questions

- View load limit (newest-N) — the design uses 25; a tunable constant, safe to
  adjust during apply without changing specs or approach.
