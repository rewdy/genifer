# Terminal preview development

Native result thumbnails are implemented under the `add-result-thumbnail`
OpenSpec change. Final interactive validation is recorded below.

## Output ownership

`internal/preview.Writer` wraps Bubble Tea's sole output writer. Private,
zero-width markers select an immutable thumbnail from a bounded two-entry
registry and its reserved cell rectangle. Markers never reach the terminal.
Native protocol output bypasses text wrapping and background painting. The
adapter clears the old image before a text write, paints the desired image after
it, and restores the cursor. Kitty deletion targets only the app's own image
ID; iTerm2 cleanup erases only the reserved rectangle. Shutdown releases images
before the renderer exits the alternate screen, without closing stdout.

## Fixture harness

Run directly in a normal terminal, outside tmux/screen/SSH:

```sh
# Ghostty
go run ./cmd/preview-check -protocol kitty

# iTerm2
go run ./cmd/preview-check -protocol iterm2
```

No API credentials, network calls, output files, or paid generation are involved.
The harness shows a 240-by-150-pixel gradient with a red rectangle. Check:

1. Image appears below the heading and above the saved-fixture details.
2. Space changes status without corrupting text or duplicating the image.
3. `p` hides the image completely; press it again to restore the image.
4. Resize narrower/shorter until it hides, then restore the window.
5. `q` and Ctrl+C return to a clean terminal without image residue.

This first harness reserves space using an 8-by-16-pixel cell estimate. If the
terminal uses different cell dimensions, pass `-cell-width N -cell-height N`.
The production layout will read actual TTY geometry in a later implementation
task; the estimate is only for the protocol fixture, not the shipped feature.

## Verification record

- Automated captured-output tests: passed for Kitty and iTerm2 (2026-10-14).
- Ghostty initial check: text-only because the output adapter hid Bubble Tea's
  terminal-file interface and prevented window-size events. The adapter now
  forwards that interface; user confirmed the Ghostty recheck looks good.
- iTerm2 visual lifecycle check: user confirmed it looks good, including
  hide/show, resize, and clean exit.

Visual checks are required before completing task 1.2. The computer-control
runtime was unavailable in the implementation session; automated output tests
do not establish actual terminal rendering or cleanup behavior.

## Initial support matrix

Direct Ghostty sessions use Kitty graphics; direct iTerm2 sessions use native
inline images. Unknown terminals, redirected output, tmux/screen, SSH, and
platforms without output-TTY pixel geometry stay text-only. No terminal input
queries are sent. Pixel dimensions must be positive and match the most recent
window cell dimensions; stale or missing geometry hides the thumbnail.

The target height is 150 terminal-reported pixels, subject to cell rounding and
available width/height. Layout reserves the header, wrapped saved path/cost/seed,
status, and result shortcuts first; a preview shrinks proportionally or is omitted
when there is no usable space. No character fallback is rendered.

SVG thumbnails were deferred by user choice after dependency review found that
the planned resvg-go binding is GPL-3.0 licensed. No SVG/WASM dependency is added;
SVG results retain the normal success screen and all file/folder actions.

## Final app fixture

To exercise the real success-screen layout with a local fixture provider:

```sh
go run ./cmd/preview-check -app
```

Run once in Ghostty and once in iTerm2. Select the fixture model, enter any prompt,
press Ctrl+S, then Enter. The fixture generates locally with no credentials or
network requests, and uses a temporary output directory removed on exit. Check
thumbnail placement, resize, `p` edit, `m` new model, `n` new image, `g` re-generate,
and quit. File/folder actions use the real OS handlers on the temporary image.
Final Ghostty app-layout check: user confirmed good. Final iTerm2 app-layout
check: user confirmed good, including placement and resize/edit/quit cleanup.
Build, full tests, vet, and strict change validation also pass.

Raster preparation rejects files over 64 MiB and image headers over 32 megapixels;
SVG and malformed files are skipped. Jobs are serialized, use a five-second
context deadline, check cancellation during I/O and between decode/scale stages,
and retain only thumbnails of at most 2048 pixels per edge. Standard raster
codec/scale CPU calls finish their current bounded operation before cancellation
is observed; deadlines do not forcibly terminate Go image codecs.
