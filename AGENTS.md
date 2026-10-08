# AGENTS.md

Guidance for AI agents and contributors working in this repo. User-facing docs
live in `README.md` and `docs/`; this file covers architecture, conventions,
and workflow.

## What genifer is

A full-screen terminal UI (Bubble Tea) for generating images via OpenRouter
image models. Single Go module, `github.com/rewdy/genifer`, Go 1.24+.

## Layout

```
main.go                       Entry point: wiring, --version, provider/dir resolution
internal/config/              config.yaml + app-owned state.json + pricing cache + value directives
internal/provider/            Provider interface (the backend seam) and shared types
internal/provider/openrouter/ OpenRouter implementation of Provider
internal/gen/                 Generation workflow: compose, run (async+cancel), save, open
internal/tui/                 Bubble Tea shell: header, picker, capability-adaptive form, view
docs/                         User docs (config schema, OpenRouter notes)
openspec/                     Spec-driven change proposals and archived specs
```

## Architecture

- **`provider.Provider` is the one seam to the outside world.** The rest of the
  app never sees provider-specific request/response shapes. Add a backend by
  implementing the interface in `internal/provider/provider.go`; wire it in
  `buildProvider` in `main.go`. Only `openrouter` exists today.
- **Dependencies flow one way:** `main` → `tui` → (`gen`, `provider`, `config`).
  `provider` and `config` have no knowledge of the TUI. Keep it that way.
- **TUI is a Bubble Tea `Model`** (`internal/tui/model.go`): `Deps` is injected
  at construction, side effects happen in `tea.Cmd`s, `Update` is a pure state
  machine over `phase`. Don't do I/O directly in `Update`/`View`.
- **Pricing and cost are decorative, never fatal.** A pricing lookup failure
  means "price unknown", not an error that blocks generation.

## Conventions

- **Errors:** wrap with `%w`. The provider layer exposes sentinel errors
  (`ErrAuth`, `ErrInsufficientCredit`, `ErrGenerationFailed`,
  `ErrReferenceImagesUnsupported`) so the UI can present actionable messages
  instead of raw strings. Reuse these rather than inventing new error text.
- **Config file ownership:** `config.yaml` is user-owned — never write to it.
  `state.json` and `pricing-cache.json` are app-owned and rewritten freely. A
  missing config is a soft fallback to defaults; a malformed config is a hard
  error (no silent fallback).
- **Config directory** is `~/.config/genifer` on every platform (honoring
  `XDG_CONFIG_HOME`), set in `config.Dir()` — not the OS-specific dir.
- **Value directives:** config strings support `{env:NAME}` and `{cmd:...}`.
  Resolution is **lazy** (a command runs only when its value is needed) and
  **single-level** (resolved values are not re-scanned). Keep both properties.
- **Context:** every provider call and the generation run honor `ctx`
  cancellation. Thread `ctx` through; don't swallow cancellation.
- **Package docs:** each package has a `doc.go` with a package comment. Update
  it when a package's responsibility changes.

## Build, test, run

Prefer separate commands over chaining.

```sh
go build ./...        # build everything
go test ./...         # run all tests
go vet ./...          # vet before finishing a change
go run . --version    # version only; does not launch the TUI
go run .              # launch the TUI (needs OPENROUTER_API_KEY)
```

Run `go test ./...` and `go vet ./...` before considering a change done, or
`just check` to run build + test + vet in one step (`just` wraps these as
`build`/`test`/`vet`/`check`). Tests live beside code as `*_test.go`; match the
existing table-driven style. Don't add tests unless the change needs them.

## Versioning

`version` in `main.go` defaults to `"dev"` and is overridden at build time:

```sh
go build -ldflags "-X main.version=$(git describe --tags --always --dirty)" .
```

When unset, `buildVersion()` falls back to Go's embedded build info (module
version for `go install module@tag`, else VCS revision). Tags are SemVer with a
`v` prefix (`v0.2.0`). Bumping the reported version means tagging, not editing a
constant.

Cut releases with `just release vX.Y.Z "<summary>"` or `just bump
[patch|minor|major] "<summary>"` (bump computes the next version from the latest
tag). These enforce the policy: they validate the version, refuse a dirty tree
or an existing tag, run the gate, create an annotated (signed when a key is
configured) tag, and print the push command rather than pushing. Two gotchas:
pass a `summary` — omitting it drops you into `$EDITOR`, which hangs in
non-interactive contexts; and `just build` omits the `-ldflags` override, so it
reports the embedded-build-info version, not the `git describe` one.

## Making changes

- Keep changes minimal and targeted; touch only what the task needs.
- Match surrounding style and existing libraries (Charm stack, `yaml.v3`); don't
  introduce new dependencies without reason.
- Non-trivial or architectural changes go through `openspec/` — see the skills
  in `.agents/skills/` (`openspec-propose`, `openspec-apply-change`, etc.).
- Record meaningful decisions and learnings in `docs/`.
