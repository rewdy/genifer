# Design

## Context

See proposal.md for motivation. The user primarily uses Ghostty and iTerm2 and explicitly prefers no thumbnail to a character-block fallback on unsupported terminals. Research checked 2026-10-14.

Current implementation uses Go 1.24.2, Bubble Tea 1.3.10 and Lip Gloss 1.1.0. `gen.Run` returns the saved path, cost, and seed rather than retaining image bytes. Successful `generatedMsg` handling already schedules history and optional auto-open commands. `resultView` displays only result text; the footer owns all actions. `View` wraps the entire body, adds padding, repaints backgrounds, and sizes a full-screen canvas. Bubble Tea's renderer skips unchanged lines and truncates oversized frames from the top. Therefore inserting a raw graphics string into the current body is unsafe: graphics need explicit geometry, cleanup, and output ordering outside text wrapping and background painting.

### Library and protocol research

| Candidate | Finding | Decision |
| --- | --- | --- |
| Existing `github.com/charmbracelet/x/ansi` v0.11.6 | Local module contains `kitty.EncodeGraphics`, `kitty.Options` including placement/deletion/direct transmission, and `ansi.ITerm2` with `iterm2.File` | Reuse these pinned encoders |
| [go-termimg](https://github.com/blacktop/go-termimg) | Multiple protocols and a widget, but its current [go.mod](https://raw.githubusercontent.com/blacktop/go-termimg/main/go.mod) requires Go 1.26 | Avoid a toolchain bump and overlapping dependencies |
| [Charm Mosaic](https://github.com/charmbracelet/x/tree/main/mosaic), [PIXterm](https://github.com/eliukblau/pixterm) | Character-cell renderers; user does not want that fallback | Do not add |
| [resvg-go](https://github.com/kanrichan/resvg-go) | Go/WASM SVG rasterization without CGO; [module](https://raw.githubusercontent.com/kanrichan/resvg-go/master/go.mod) declares Go 1.24 and Wazero | GPL-3.0 binding; user chose to defer SVG thumbnails |
| [oksvg](https://github.com/srwiley/oksvg) | Explicitly partial SVG implementation | Avoid as the default for generated SVG artwork |

[Ghostty documents Kitty graphics support](https://ghostty.org/docs/about). [iTerm2 documents its inline protocol](https://iterm2.com/documentation-images.html), including pixel sizing and aspect preservation; current docs also mention Kitty support, but native inline images cover more iTerm2 installations. Use separate adapters rather than assuming every iTerm2 version implements Kitty. The [Kitty protocol](https://sw.kovidgoyal.net/kitty/graphics-protocol/) supports identified image placements and deletion.

## Goals / Non-Goals

**Goals:** Native graphics with reliable result-screen lifecycle, approximately 150 terminal pixels high, no changes to provider/generation APIs, and zero user-visible failure when preview support is unavailable.

**Non-Goals:** SVG rasterization, full-screen viewing, zoom, animation, history thumbnails, reference-image previews, sixel, character mosaics, protocol selection UI, new config settings, a Bubble Tea upgrade, tmux/screen passthrough, or remote-session support in this first change.

## Decisions

### 1. Keep preview work independent of generation

Add `internal/preview` for local decoding, fitting, terminal capabilities, and protocol encoding. It must not import `tui` or know provider response shapes. Read the successful saved path in a cancellable command; leave `gen.Outcome` unchanged. Add an injected preview service/terminal geometry seam to TUI dependencies for deterministic tests. Update relevant package comments.

The preview never changes the saved file, never creates a permanent sidecar, and never invokes a provider or uploads an image. Start it alongside history/auto-open, not before showing success. Show no loading placeholder. If preparation fails, retain the same success message and shortcuts.

Use one bounded preview job at a time, identified by result revision and geometry revision. Cancel on navigation, replacement, resize, and exit. Ignore late completions, including cancelled decoding. Retain only the active result's bounded prepared thumbnail/working state; no persistent preview cache. A resize can re-read the same saved file in a new bounded command rather than retaining a large original raster indefinitely.

### 2. Conservative terminal support and geometry

Require an actual output TTY and a known direct terminal session: `TERM_PROGRAM=ghostty` selects Kitty, `TERM_PROGRAM=iTerm.app` selects iTerm2 inline. Reject `TERM=dumb`, tmux/screen indicators, and SSH indicators before emitting any graphics. These are conservative environment-based selections, not universal feature detection. Unknown terminals remain text-only. Do not run terminal queries that compete with Bubble Tea's input reader or produce startup delays.

Get terminal window pixel dimensions and cell counts from the output TTY through a platform-specific, injected geometry reader using existing `x/sys` facilities on supported Unix systems. Refresh in a command on resize; Bubble Tea v1's `WindowSizeMsg` alone only supplies cells. Derive cell width/height from positive pixel dimensions and matching cell counts. If metrics are zero, stale, or inconsistent, omit the thumbnail until usable metrics arrive. Do not promise exact physical pixels across Retina scaling; 150 means terminal-reported pixels, rounded to cell boundaries.

Unsupported platform readers return unavailable rather than preventing build or launch. This initial matrix deliberately prefers hiding a possible preview to sending unknown escape sequences.

### 3. Fit within the actual result layout

Compute a pure layout plan from header height, wrapped saved path/cost/seed, wrapped footer/status, content width, and terminal geometry. Reserve all text and existing spacing first. Put the left-aligned preview before saved-result text. Its preferred pixel height is 150; proportional width follows source dimensions. Apply a uniform scale limited by available pixel width and remaining pixel height, then reserve the ceiling number of rows/columns without stretching the prepared bitmap to a different aspect ratio. Round dimensions inward to avoid overflow.

If no whole usable row/column remains, render the original success layout with no added space. Recalculate when status wraps differently as well as on resize. Text-only frames remain byte-equivalent to the current layout for the same state. Do not solve overflow by letting Bubble Tea drop the header.

### 4. Serialize native graphics with Bubble Tea output

Use `tea.WithOutput` in `Run` to wrap the normal output writer in a small preview-aware adapter. Keep `View` pure: it renders ordinary text and blank preview rows, then inserts a private zero-width placement/clear marker into the first physical line after all wrapping, padding, and painting, with a duplicate at the end of the frame so clear markers survive top-clipping in tiny terminals. The marker references prepared data by opaque ID and includes its layout rectangle; it contains no file contents or terminal image payload. A bounded runtime registry is populated by preparation commands, not by `View`.

The output adapter consumes markers rather than passing them to the terminal. It handles split writes, tracks the active rectangle/ID, clears the previous graphic before writing a changed text frame, writes Bubble Tea's text unchanged, then draws/repositions the active thumbnail using cursor save/restore. Subsequent renderer writes reuse the desired preview state when an unchanged marker line is skipped. A changed or cleared marker must change that header line, ensuring navigation reaches the adapter even when other text matches. No concurrent raw stdout writes from commands.

For Kitty, use a dedicated positive image/placement ID, direct chunked transmission, quiet mode suppressing responses, cursor-preserving placement, and delete only app-owned resources. Never issue delete-all. For iTerm2, use an inline PNG payload with proportional fit and explicit bounded dimensions; erase the prior reserved cell rectangle before the next text frame so obsolete inline images cannot survive. Validate this erase/repaint behavior in real iTerm2 during the first integration task. Do not rely on a standalone `imgcat` process.

Adapter shutdown clears its graphic before alternate-screen teardown and releases registry entries on normal quit and errors; it is idempotent and does not close the caller's stdout. Protocol byte strings bypass Lip Gloss and ANSI wrapping entirely. This approach retains Bubble Tea v1's text renderer rather than replacing the framework or racing its output goroutine.

### 5. Bounded raster and SVG preparation

Decode PNG/JPEG/WebP (and a first GIF frame when encountered) with Go image decoders. Inspect dimensions before full decode. Use source file limit 64 MiB, raster dimension budget 32 megapixels, thumbnail edge cap 2048 pixels, and a five-second preparation deadline. These limits only suppress the optional preview; they never change generation success. Scale with the already present `x/image/draw` and encode a thumbnail PNG for both protocols. Preserve alpha.

SVG rasterization is deferred by explicit user choice during implementation: the researched resvg-go binding is GPL-3.0 licensed. Do not add it or Wazero. SVG results keep the normal success layout and all existing actions, without an attempted raster decode or warning. A future change can evaluate a different renderer. The first release adds no image rendering dependency.

## Risks / Trade-offs

- [Native graphics can survive ordinary text redraws] → explicit owned placement/rectangle cleanup and real-terminal navigation/resize tests are required early, before full feature integration.
- [An output adapter touches a sensitive rendering boundary] → deterministic captured-output tests for marker stripping, chunk boundaries, partial writes, repeated frames, shutdown, and cursor restoration; preserve all unrelated output.
- [Environment detection can miss supported terminals] → conservative text-only fallback; document the supported direct-session matrix and metric requirement.
- [SVG previews are absent in this release] → document the deliberate raster-only scope and keep existing SVG open/folder actions.
- [Headless tests cannot prove graphics cleanup] → a fixture-only interactive harness must be checked in Ghostty and iTerm2; no paid generation is necessary.

## Migration Plan

Implement and manually validate the output adapter with a tiny fixture first, then geometry/fitting, image preparation, and result integration. If either target terminal cannot pass cleanup/positioning checks with the selected protocol, stop and revise the design before proceeding; do not silently ship one-terminal support or a character fallback. Add user-facing limitations and manual QA results to documentation. Run build/test/vet and strict OpenSpec validation. No data migration; rollback reverts preview wiring/dependencies while leaving saved images and result actions intact.
