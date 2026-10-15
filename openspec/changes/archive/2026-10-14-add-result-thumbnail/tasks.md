# Tasks

## 1. Native terminal output integration

- [x] 1.1 Add the preview protocol/output adapter using the existing pinned Charm Kitty and iTerm2 encoders, opaque placement markers, and a bounded prepared-image registry. Verify captured-output tests cover byte preservation, marker stripping across split writes, quiet Kitty commands, owned-ID deletion, iTerm2 inline PNG dimensions, cursor restoration, partial writes/errors, unchanged marker rows, and idempotent shutdown before alternate-screen exit. Add the new package's `doc.go` and document output ownership.
- [x] 1.2 Add a fixture-only Bubble Tea v1 harness for both protocols and record manual checks in `docs/terminal-previews.md`: initial render, status redraw, movement between screens, resize, and exit in Ghostty and iTerm2. Verify no image residue or text corruption. Stop and revise the design if either target fails rather than silently dropping its support; no paid generation is needed.

## 2. Support detection and proportional fitting

- [x] 2.1 Add injected terminal detection and platform-specific output-TTY pixel geometry readers with conservative unsupported fallbacks. Verify tests for Ghostty/iTerm2 selection, unknown terminals, redirected output, `TERM=dumb`, tmux/screen, SSH, zero metrics, and mismatched/stale resize geometry. Document the initial direct-session support matrix and geometry requirement in `docs/terminal-previews.md`.
- [x] 2.2 Add pure fitting/layout helpers that target 150 terminal pixels high and reserve all header, wrapped result details, status, and footer rows first. Verify table-driven landscape/portrait/square/extreme-ratio cases, narrow/short terminals, cell rounding, long saved paths, changing footer heights, and omission when no area remains; verify no stretching/cropping or header clipping.

## 3. Local image preparation

- [x] 3.1 Add cancellable, bounded PNG/JPEG/WebP preparation and scaling from the saved file to an in-memory PNG, preserving alpha. Verify image fixtures, invalid/missing files, file/dimension limits, deadline/cancellation behavior, proportional dimensions, and byte-identical original files. Document preparation limits and best-effort behavior alongside the package API.

## 4. Success-screen lifecycle

- [x] 4.1 Wire the preview service through TUI dependencies and Run's output adapter; prepare previews alongside existing success commands, attach results only for the current result/geometry revision, and render reserved space only after a usable preview is ready. Verify model tests for every result action, history/auto-open independence, failed outcomes, late completions after navigation/new results, resize cancellation/refit, status redraws, and quit cleanup. Update TUI package docs and README with the automatic thumbnail behavior and silent fallback.
- [x] 4.2 Add integrated fixture-based success-screen tests for raster output and text-only SVG results, both native protocols, missing geometry, and unsupported sessions; verify saved path/cost/seed and all existing shortcuts remain visible and functional, no graphics reach text styling, and unsupported sessions retain the original success layout. Repeat the Ghostty/iTerm2 harness lifecycle checks with final layout and record results.

## 5. Final integration gate

- [x] 5.1 Run `go build ./...`, `go test ./...`, `go vet ./...`, and `openspec validate add-result-thumbnail --strict` as separate commands. Verify success, review final dependency/toolchain changes, and confirm no config/state/provider contract or saved-file changes.
