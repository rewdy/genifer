# Proposal

## Why

Prompts are lost the moment a session ends. A user who wants to re-run or adapt
something they generated yesterday must retype it from memory. `state.json`
remembers only the last model and aspect ratio; there is no record of what was
generated. A local, reviewable history lets users recall and reuse past prompts
without external note-taking. History is stored on-device only and is never
transmitted anywhere; it is a convenience, enabled by default, that the user can
opt out of.

## What Changes

- Persist a record of each successful generation to an app-owned history cache
  in the config dir (prompt, model, aspect ratio, the seed used, saved image
  path, timestamp). The seed is recorded whenever one was used so a past
  generation can be reproduced (reuse the seed) or deliberately varied (change
  it). Following the pricing-cache pattern, the file is versioned and
  best-effort: a missing or unparsable file is not an error.
- Add a lightweight in-tool history view that lists past prompts newest-first
  and lets the user pick one to reuse (populating the compose screen with its
  prompt and recoverable parameters).
- Retain all history records — nothing is pruned or deleted. The history file
  is small (prompt text plus metadata), so unbounded growth is not a concern,
  and clearing history is simply deleting the file.
- Keep the UI performant by loading only the most-recent entries into the
  history view. When the user pages past the loaded entries (and older ones
  exist on disk) or filters to no matches, point them to the on-disk history
  file where the full record lives, rather than building paging UI for the
  long tail — recall is almost always of something recent.
- Add a config option to opt out of history. History is enabled by default;
  when disabled, nothing is written and the history view reports that it is off.

## Capabilities

### New Capabilities
- `prompt-history`: persisting a bounded, versioned record of successful
  generations, reviewing them in a newest-first list, reusing an entry to
  pre-fill compose, and opting out via config.

### Modified Capabilities
<!-- None: no pre-existing specs. The history opt-out adds a config field but
     there is no existing config capability spec to modify. -->

## Impact

- `internal/config/`: a new `history-cache.json` (app-owned) with load/save
  following `pricing.go`; a new `HistoryEnabled` config field (default true) in
  `config.go`.
- `internal/tui/model.go` + `view.go`: a new history phase/view, a key to open
  it, selection-to-compose wiring, and a write-on-success hook in the
  `generatedMsg` handler.
- `internal/gen/` (read-only usage): the saved path and parameters needed for a
  history record already flow through `Outcome`/`Draft`.
- `docs/config.md`: document the `history` option and the cache file.
