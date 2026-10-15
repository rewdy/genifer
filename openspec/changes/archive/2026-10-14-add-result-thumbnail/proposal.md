# Proposal

## Why

The success screen currently requires opening a separate viewer to see what was generated. A small native image preview lets users assess the result before editing, changing models, or generating again.

## What Changes

- Show an automatic, left-aligned thumbnail above saved-path/cost/seed details on successful generation, targeting 150 terminal pixels high with proportional width.
- Support direct Ghostty and iTerm2 sessions; hide the thumbnail completely when terminal support, usable pixel geometry, or rendering is unavailable. No character-block fallback or empty placeholder.
- Fit the preview within available width and height while preserving the existing result actions, header, and footer.
- Prepare PNG/JPEG/WebP previews asynchronously from the saved local file. Keep original output files unchanged and all preview failures non-fatal.
- Clear native image placements when navigating, regenerating, resizing, or quitting; ignore stale background results.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `post-generation-actions`: add optional result thumbnails and their sizing, fallback, and lifecycle behavior.

## Impact

Changes are centered on `internal/tui` result state, layout, commands, and `Run` output wiring, plus a small internal preview package and documentation/tests. SVG thumbnails are deferred by explicit user choice after discovering the planned resvg-go binding uses GPL-3.0. Reuse the pinned `charmbracelet/x/ansi` Kitty/iTerm2 encoders and existing `x/image` scaling/decoding. No SVG renderer dependency is added. No provider API, saved-file, config, history, or Go toolchain migration. Initial support excludes tmux/screen passthrough and SSH sessions; these use the normal success screen.
