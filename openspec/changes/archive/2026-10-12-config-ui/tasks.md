# Tasks

## 1. Write settings back to config.yaml

- [x] 1.1 Add a config writer that updates `output_dir`, `auto_open`, and the history key in an existing `config.yaml`, preserving comments and other settings (targeted key edit, or `yaml.Node` round-trip), and writes atomically (temp file + rename); verify a test asserts the three keys change while comments and the `api_key` directive survive and the file still `Load`s.
- [x] 1.2 Add a save-location validation helper (resolve via `ResolveOutputDir` + creatable/writable probe) used before any write; verify tests for an accepted path and a rejected path, with the rejected path leaving the file unchanged.

## 2. Live application of settings

- [x] 2.1 Make the effective output dir mutable for the session so a changed save location applies to the next `generate()`; verify a test asserts the next generation targets the updated directory.
- [x] 2.2 Update the in-memory `Config` (auto-open, history) on confirm so changes take effect without restart; verify a test asserts a toggled auto-open changes the open behavior in-session.

## 3. Config view

- [x] 3.1 Add a `phaseConfig` and a config view with a save-location field and auto-open (and history) toggles showing current values from the loaded `Config`; verify a view test asserts current values render.
- [x] 3.2 Add a key to open the config view, with `esc` to return and a confirm action that validates, writes `config.yaml`, and updates the in-memory config; verify a test drives open → edit → confirm → written-and-returned.
- [x] 3.3 On a rejected save-location value, report the problem and keep the prior value without writing; verify a test asserts no write occurs on invalid input.

## 4. Integration and docs

- [x] 4.1 Update `docs/config.md` to note that the config view edits `config.yaml` directly; verify the documented editable keys match the writer.
- [x] 4.2 Run `go build ./...`, `go test ./...`, and `go vet ./...` (or `just check`) and verify all pass.
