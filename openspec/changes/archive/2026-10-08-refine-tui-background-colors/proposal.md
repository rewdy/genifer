# Proposal

## Why

The TUI already paints an app background (`#1e1c32`) on the header and on the
full-screen canvas, but the embedded widgets — the model picker list and the
prompt textarea — render with their own defaults. The result is that the
terminal's default background shows through in the body: bands of the app
background alternate with bands of whatever the user's terminal uses, so the UI
reads as patched together rather than as one surface.

## What Changes

- Apply the app background color to every widget that renders into the body, so
  the picker list, prompt textarea, and their rows are consistently painted
  instead of falling back to the terminal default.
- Fill each rendered row to the full width with the app background so short
  lines (list rows, prompt lines, status lines) do not leave trailing cells in
  the terminal's own color.
- Centralize the background color and the derived styles in one place in the
  `tui` package, so new widgets pick up the shared surface by default rather
  than each defining its own styling.
- Preserve the existing palette: the header wordmark, pink/purple accents,
  selection highlights, and pills keep their current colors; only the
  surfaces behind them become consistent.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `tui`: The full-screen application shell requirement gains an explicit
  expectation that the app background covers the entire surface uniformly,
  including embedded widgets, with no terminal-default background showing
  through.

## Impact

- `internal/tui/view.go`, `internal/tui/model.go` (widget construction),
  `internal/tui/header.go` (shared color/style source), and
  `internal/tui/delegate.go` (list row painting).
- No provider, config, or generation behavior changes. No new dependencies.
