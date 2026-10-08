# Proposal

## Why

A `~`-prefixed `output_dir` (e.g. `~/Downloads/genifer`) is not expanded. The leading `~` is not an absolute path, so the value is treated as relative and joined to the launch directory, writing images into a literal `~` directory under the current working directory instead of the user's home.

## What Changes

- Expand a leading `~` / `~/...` in `output_dir` to the user's home directory during output-dir resolution, so `~/Downloads/genifer` resolves to an absolute path under `$HOME`.
- A bare `~` resolves to the home directory itself.
- No change to existing unset / `.` / `pwd` / relative / absolute behavior.

## Capabilities

### New Capabilities

<!-- none -->

### Modified Capabilities

- `config`: the "Output directory resolution" requirement adds `~` / `~/...` home-directory expansion as a resolution case.

## Impact

- `internal/config/outputdir.go` (`ResolveOutputDir`): add tilde expansion.
- A tilde-expansion helper exists in `internal/tui/refimage.go` (`expandHome`); the config layer needs equivalent behavior without depending on the TUI package (dependencies flow `tui` → `config`, not the reverse).
- `internal/config/outputdir_test.go`: add cases for `~` and `~/...`.
- No config-file format change; existing configs keep working.
