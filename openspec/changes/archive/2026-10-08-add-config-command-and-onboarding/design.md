# Design

## Context

See `proposal.md` — Why. The app today never writes `config.yaml`
(`internal/config/config.go`): `Load` soft-falls-back to `Default()` with
`ErrConfigNotFound`, and `main.go` only prints a stderr note before launching
the TUI. Argument handling in `run()` is a flat scan for `version`/`-v`. The
TUI (`internal/tui`) is a Bubble Tea `Model` with a `phase` state machine
(`phasePicker` … `phaseResult`) and `Deps` injected at construction via
`tui.Run`. Config strings are `config.Value` with `{env:}`/`{cmd:}` directives
(`internal/config/resolve.go`). Config dir is `config.Dir()`; path is
`config.ConfigPath()`.

Two surfaces are affected: a non-TUI CLI command path (`genifer config …`) and
a new TUI onboarding phase. Both need the same new primitive — write a commented
starter `config.yaml` only when absent — so that primitive is the shared seam.

## Goals / Non-Goals

**Goals:**
- A `config.Value`-aware starter-file writer that is create-only and produces
  valid, loadable, commented YAML.
- `genifer config` (create-if-missing + open in resolved editor) and
  `genifer config path` (print path, no side effects) as subcommands dispatched
  from `main.go`.
- An in-TUI first-run onboarding phase that captures the API-key provider choice
  (paste / env / command) and writes it as the matching directive.
- Preserve the user-owned invariant: never overwrite an existing `config.yaml`.

**Non-Goals:**
- No `config get`/`config set`, no config validation command, no interactive
  settings editor beyond the onboarding key prompt.
- No new third-party dependencies.
- Onboarding does not verify the key against the provider; it only records the
  choice.

## Decisions

**1. Shared starter writer lives in `internal/config`.**
Add a function (e.g. `WriteStarter(path string, apiKey Value) (bool, error)`)
that returns `false` without writing when the file exists, and otherwise creates
the dir and writes a commented template with the given API-key directive
substituted (defaulting to `{env:OPENROUTER_API_KEY}`). Rationale: both the
command and the TUI need identical create-only semantics; centralizing keeps the
"never overwrite" rule in one place and keeps `tui`→`config` dependency
direction intact (AGENTS.md: `provider`/`config` never depend on `tui`).
*Alternative considered:* generate the file in `main.go` — rejected, it would
duplicate the create-only guard across the command and onboarding.

**2. Starter content is a hand-written commented template, not a marshaled
`Default()`.** `yaml.Marshal` of the struct would drop comments and emit zero
values, which is poor onboarding. A literal template string documents each key
with its default. Rationale: the whole point is a self-documenting file. The
template is covered by a test that loads it back through `config.Load` to prove
it stays valid as keys evolve.

**3. CLI dispatch stays minimal, extended in `run()`.** Replace the flat flag
scan with handling for the first positional: `version`/`-v`/`--version` (as
today), and `config` with an optional `path` sub-token. `genifer config path`
prints `config.ConfigPath()` and returns; `genifer config` resolves the editor,
calls the starter writer, then execs the editor and returns — never reaching
`tui.Run`. Rationale: matches the existing bareword-subcommand style without
pulling in a CLI framework (no new dependency).

**4. Editor resolution: `$VISUAL` → `$EDITOR` → OS default.** A small helper
(in `internal/config`, alongside the existing OS-default open logic the app
already uses for images, but kept distinct from `open_command`, which is for
image viewers). OS default: `open -t` on macOS, `xdg-open`/`editor` on Linux,
`notepad` on Windows — but prefer a real editor env var first. If none resolve,
print the path and an actionable error (spec: "No editor available"). The editor
runs with inherited stdio so interactive editors work.

**5. Onboarding is a new leading TUI phase, gated on first-run detection in
`main.go`.** `main.go` already knows `noConfig`. Pass that into `Deps` (e.g.
`Deps.FirstRun bool` plus `Deps.ConfigPath string`) and start the model in a new
`phaseOnboarding` before `phasePicker` when set. The onboarding phase renders
the WELCOME banner and a choice list (paste / env / command); selecting one
collects the follow-up input (reusing the existing `textarea`/text input
patterns) and, on completion, calls the shared starter writer with the chosen
`config.Value` (literal, `{env:NAME}`, or `{cmd:...}`), then transitions to
`phasePicker`. Rationale: keeps onboarding inside the Bubble Tea state machine
(no pre-TUI stdin prompt), consistent with the architecture note that side
effects happen in `tea.Cmd`s. The file write is a `tea.Cmd`, not inline in
`Update`.

**6. Writing the key directive maps directly onto `config.Value` forms.** paste
→ literal string; env → `{env:NAME}`; command → `{cmd:...}`. No new encoding.
The paste path shows the plaintext-storage warning and requires confirmation
before writing (spec scenario).

## Risks / Trade-offs

- **Interactive editor vs. alt-screen TUI** → The `config` command path never
  enters the alt screen (it returns before `tui.Run`), so there is no terminal-
  state conflict; the editor owns the terminal normally.
- **First-run write failure mid-onboarding** (e.g. unwritable config dir) →
  surface an actionable error in the onboarding phase and still allow continuing
  on in-memory defaults for the session, rather than hard-exiting; the invariant
  (never overwrite) is unaffected because we only ever create.
- **Starter template drifts from the real schema as config fields are added** →
  Mitigated by a test that loads the generated starter through `config.Load`;
  AGENTS.md already asks that `doc.go`/docs track schema changes.
- **OS-default editor guesswork is imperfect across environments** → `$VISUAL`/
  `$EDITOR` take precedence, and the "no editor" path degrades to printing the
  path, so the command is never a dead end.

## Open Questions

- Exact visual treatment of the WELCOME banner (ASCII art vs. styled text) is a
  presentation detail left to implementation; it does not affect the spec or
  task breakdown.
