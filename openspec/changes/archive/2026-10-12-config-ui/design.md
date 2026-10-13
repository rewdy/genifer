# Design

## Context

See proposal.md — Why. `Config` already holds `OutputDir Value`,
`AutoOpen bool`, and (via the prompt-history change) `HistoryEnabled`.
`config.yaml` is a hand-commented YAML file: it documents every option, uses
directive values like `api_key: "{env:OPENROUTER_API_KEY}"`, and leaves optional
keys (`output_dir`, `open_command`) commented out (see
`internal/config/starter.go`). Until now the app only ever *created* this file
(onboarding/starter) and never edited an existing one. Output dir is resolved in
`main.resolveOutputDir`/`config.ResolveOutputDir` and injected as
`Deps.OutputDir`; `AutoOpen` is read live in the `generatedMsg` handler.

The project decision for this change: the config view is a graphical editor
**for `config.yaml` itself**. No override file, no precedence layer — edit the
file, write it back.

## Goals / Non-Goals

**Goals:**
- An in-tool config view for exactly three settings that writes back to
  `config.yaml`.
- Keep the user's other settings intact across a write.
- Apply save location and auto-open live for the running session.

**Non-Goals:**
- Editing every config field (API key, provider, base URL, open-command) from
  the UI — those stay file/onboarding-managed.
- Resolving `{env:}`/`{cmd:}` directives from the UI; the view edits literal
  setting values, not directives.
- Perfectly round-tripping every comment and formatting nuance of the file (see
  Risks).

## Decisions

- **Edit the file directly; no override store.** Reverses the earlier
  override-in-`state.json` idea per project direction. The three settings are
  written into `config.yaml`; load-time precedence stays as it is today
  (`Default()` → file). Rationale: this is a small single-user tool and the view
  is explicitly a graphical editor of the file.
- **Write strategy that preserves unrelated settings.** The three editable
  settings (`output_dir`, `auto_open`, and the history key) are simple scalars.
  Prefer a **targeted update** of just those keys rather than a full
  `yaml.Marshal` of the `Config` struct, because a full re-marshal would drop
  the file's comments and rewrite the `api_key` directive. Options, cheapest
  first:
  1. **Set/replace the specific keys in the existing text**, appending a key if
     it is absent (e.g. a commented-out `output_dir`). Preserves comments and
     all other content; the trade-off is simple scalar-key handling only, which
     is all three settings need.
  2. **`yaml.Node` round-trip** (yaml.v3 supports comment-preserving nodes) to
     edit the parsed tree and re-encode. More robust for structure, more code.
  Decision: start with targeted key update (option 1) since the editable set is
  three top-level/one-nested scalar(s); fall back to `yaml.Node` only if that
  proves fragile. Either way, the write MUST be followed by a `Load` to confirm
  the file still parses (covered by a test).
- **Validate save location before writing.** Resolve via `ResolveOutputDir` and
  probe that the directory is creatable/writable (`MkdirAll` + write check); on
  failure keep the prior value, show the error, and do not touch the file.
- **Apply live.** Make the session's effective output dir mutable so a new save
  location is used by the next `generate()` without restart; auto-open is
  already read at generation time, so updating the in-memory `Config.AutoOpen`
  (and history flag) suffices.
- **Config view as a new `phaseConfig`** with a save-location text field and
  auto-open/history toggles, consistent with the aspect-ratio modal / onboarding
  input idioms. A key opens it; `esc` returns; confirm validates, writes
  `config.yaml`, and updates the in-memory config.

## Risks / Trade-offs

- [Comment/formatting preservation] → A naive full re-marshal would erase the
  file's helpful comments and the `api_key` directive. Mitigation: targeted
  key-level edit (or `yaml.Node`), plus a mandatory re-`Load` after writing and
  a test asserting comments and the API-key directive survive a round-trip.
- [First app-driven write to a user-owned file] → Previously only created via
  `O_EXCL`. Now edited in place. Mitigation: write atomically (temp file +
  rename) so an interrupted write cannot corrupt the config; validate by
  re-loading.
- [Malformed existing file] → `Load` treats a malformed file as a hard error, so
  the app would not have reached the TUI with one. The editor can therefore
  assume a parseable starting file.

## Open Questions

- Whether to adopt the `yaml.Node` approach up front vs the lighter targeted
  edit — decidable during apply by trying the targeted edit against the starter
  template and the test suite; both satisfy the spec.
