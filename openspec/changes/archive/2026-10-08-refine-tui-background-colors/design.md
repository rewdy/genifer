# Design

## Context

See proposal.md — Why. The shell already has a shared background constant,
`headerBG` (`#1e1c32`), in `internal/tui/styles.go`. `Renderer`-level surfaces
(the header, the full-screen canvas in `Model.View`) paint it, but the two
embedded widgets do not:

- **Prompt textarea** (`bubbles/textarea`). genifer leaves `FocusedStyle` /
  `BlurredStyle` at their defaults. The default `CursorLine` background is
  `AdaptiveColor{Light: "255", Dark: "0"}`, which paints the cursor row with a
  hardcoded black rather than the app background.
- **Model list** (`bubbles/list`). genifer styles the item text through
  `modelDelegate`, which paints `headerBG` on the glyphs it emits but truncates
  rows without filling them to the width. The list's own `Styles` (filter
  prompt, pagination dots) are left at defaults and carry no background.

Both widgets therefore emit cells with no background (terminal default) or with
a conflicting background, so the terminal's default shows through as bands —
exactly the symptom in the reference screenshot. The full-screen canvas pads the
*whole* content block with `headerBG`, but it cannot repaint cells an inner
widget already emitted with a different color, and it does not reach blank
filler rows inside a widget.

Constraints: `internal/tui` may only depend on the packages it already imports
(lipgloss, bubbles, bubbletea); no provider/config/generation changes; the
existing palette must be preserved.

## Goals / Non-Goals

**Goals:**
- Every cell the TUI renders — including inside embedded widgets and on blank
  filler rows — carries the app background, so the terminal default never shows
  through.
- One place in the package defines the background and the derived widget
  styles, so future widgets inherit the shared surface by default.
- Rows shorter than the content width are filled to the width with the app
  background.

**Non-Goals:**
- Changing any color other than background coverage (wordmark pink/purple,
  accents, selection, pills are unchanged, though they may gain the shared
  background behind them).
- Making the background user-configurable or themeable (out of scope for this
  change).
- Touching the alt-screen behavior or the exit path.

## Decisions

### 1. Centralize background + derived styles in one `styles.go`

Move the shared background constant and add package-level styles there:
`appBackground` (the color), and prebuilt `lipgloss.Style` values for the
surfaces the package needs (canvas base, list styles, textarea styles). Keep
the existing `headerBG` identifier (or alias it) so `header.go` and
`delegate.go` keep compiling and the palette stays single-sourced.

*Alternative considered:* leave colors scattered and just patch each widget.
Rejected — the bug is precisely that each render site decides its own
background; centralizing prevents recurrence.

### 2. Give the textarea explicit focused/blurred styles over the input background

After `textarea.New()`, set `FocusedStyle` and `BlurredStyle` so `Base`,
`CursorLine`, `Text`, `Placeholder`, `Prompt`, and `EndOfBuffer` all share the
input background — a shade lighter than the app background — and a foreground
consistent with the body text. This removes the default `CursorLine` black band
and makes the prompt read as a distinct input area. The picker's filter field
uses the same input shade. The textarea's placeholder render path does not pad
its lines, so `composeView` also paints the rendered prompt over `inputBG`
before embedding it — filling the whole input box, not just the placeholder
glyphs.

### 3. Give the list explicit styles over the app background

Set `picker.Styles` so `FilterPrompt`, `FilterCursor`, `PaginationStyle`,
`ActivePaginationDot`, `InactivePaginationDot`, `NoItems`, and the status/filter
styles all paint over the app background. This covers the filter line and the
pagination dots, which otherwise show through.

### 4. Fill every delegate row to the list width with the app background

In `modelDelegate.Render`, after composing the (already `headerBG`-painted)
line, pad it to `m.Width()` with app-background-styled spaces instead of only
truncating. This removes trailing terminal-colored cells on short rows.

### 5. Paint the composed view line-by-line as a final backstop

Add a small helper that splits the composed view on `"\n"` and pads each line to
`m.width` with app-background spaces, then joins. This catches blank filler rows
inside the list viewport and any future widget that forgets to paint. Apply it
in `Model.View` in place of (or in addition to) the canvas `Width`/`Height`
padding. This is a backstop, not a substitute for decisions 2–4: it can only
fill short lines, not repaint a cell an inner widget already colored.

## Risks / Trade-offs

- [Over-aggressive padding could clip or wrap content] → Pad only to the known
  display width using `lipgloss.Width` / `ansi.StringWidth` for measurement,
  never by byte length, and never pad a line already at or over the width.
- [Duplicating style setup between focused and blurred textarea styles drifts]
  → Build both from one helper in `styles.go`.
- [Tests that inspect rendered output may assert on exact escapes] → Existing
  tests assert on substrings (e.g. wordmark colors, widths); re-run
  `go test ./...` and adjust only if a test encodes the old default-background
  behavior.
- [Reset/no-color terminals] → Colors are still emitted as before; a terminal
  without color support ignores them, so behavior degrades to today's.
