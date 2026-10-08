# Tasks

## 1. Starter config writer (shared seam)

- [x] 1.1 Add a commented starter `config.yaml` template string in `internal/config` documenting every key with its default; verify `go build ./...` succeeds.
- [x] 1.2 Implement a create-only writer (e.g. `WriteStarter(path string, apiKey Value) (created bool, err error)`) that returns `created=false` without writing when the file exists, else creates the config dir and writes the template with the given API-key directive substituted (default `{env:OPENROUTER_API_KEY}`); verify with a unit test covering the write-when-absent and skip-when-present cases.
- [x] 1.3 Add a test that loads the generated starter back through `config.Load` and asserts it parses as valid config with no error; verify `go test ./internal/config/` passes.

## 2. Editor resolution

- [x] 2.1 Add an editor-resolution helper in `internal/config` returning the editor command via `$VISUAL` → `$EDITOR` → OS default (`open -t` on macOS, `xdg-open`/`editor` on Linux, `notepad` on Windows), kept distinct from `open_command`; verify with a table-driven unit test over env combinations and GOOS defaults.
- [x] 2.2 Signal the no-editor-resolvable case so callers can print the path and an actionable error; verify the helper's test asserts that case.

## 3. `genifer config` CLI commands

- [x] 3.1 Extend `run()` dispatch in `main.go` to handle the first positional `config`, with an optional `path` sub-token, alongside existing `version` handling; verify `go build ./...` succeeds.
- [x] 3.2 Implement `genifer config path` to print the resolved `config.ConfigPath()` and return without creating the file, opening an editor, or launching the TUI; verify by running `go run . config path` and confirming it prints the absolute path and exits (file untouched).
- [x] 3.3 Implement `genifer config` to call the starter writer (create-if-missing), resolve the editor, and exec it on the config file with inherited stdio, returning before `tui.Run`; on no resolvable editor, print the path and an actionable error. Verify by running `EDITOR=true go run . config` on a temp config dir (file created, editor invoked, TUI not launched).
- [x] 3.4 Document the `config` and `config path` commands in `docs/config.md` and `README.md`; verify the documented commands run as written.

## 4. First-run onboarding TUI flow

- [x] 4.1 Thread first-run state into the TUI: add `FirstRun bool` and `ConfigPath string` to `tui.Deps`, populate them from `noConfig`/`cfgPath` in `main.go`; verify `go build ./...` succeeds.
- [x] 4.2 Add a leading `phaseOnboarding` phase to the TUI model that renders the WELCOME banner and an API-key provider choice list (paste / env / command), shown only when `FirstRun` is set, otherwise starting at `phasePicker`; verify with a model unit test asserting the initial phase for both first-run and config-present cases.
- [x] 4.3 Collect the follow-up input per choice (env var name, command, or pasted literal) using the existing text-input patterns, and for paste show the plaintext-storage warning requiring confirmation; verify with a unit test driving each branch to its collected value.
- [x] 4.4 On completion, run a `tea.Cmd` that calls the shared starter writer with the chosen `config.Value` (literal / `{env:NAME}` / `{cmd:...}`) and transitions to `phasePicker`; on write failure surface an actionable error and continue on in-memory defaults. Verify with a unit test asserting the correct directive is written for each choice and that an existing config is never overwritten.
- [x] 4.5 Document the first-run onboarding flow in `README.md`; verify the described behavior matches the implemented phases.

## 5. Integration verification

- [x] 5.1 Run `go build ./...`, `go test ./...`, and `go vet ./...`; verify all pass.
- [x] 5.2 Manual end-to-end against a temp `XDG_CONFIG_HOME`: launch with no config and complete onboarding (each key-provider choice writes the right directive and lands in the picker); re-launch and confirm onboarding is skipped; run `genifer config path` and `genifer config`. Verify observed behavior matches the config and tui specs.
