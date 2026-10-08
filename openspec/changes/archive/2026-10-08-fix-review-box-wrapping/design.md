# Design

## Context

See proposal.md — Why. Current state:

- `reviewView` (`internal/tui/view.go`) builds the box with `lipgloss.NewStyle().Border(RoundedBorder).Padding(0,1).Render(prompt)` and sets **no** `.Width(...)`. Lipgloss therefore sizes the border to the natural width of the content's longest physical line. A long prompt is one logical line that the terminal draws wider than the viewport, so the rendered border is both mis-sized and overflowed — the symptom in the screenshot.
- The compose prompt already bounds itself: on `tea.WindowSizeMsg`, `m.prompt.SetWidth(max(20, msg.Width-4))`. `m.width` is stored on the model and is the terminal width.
- `gen.Draft.Review()` just trims trailing newlines; it does not wrap.

## Goals / Non-Goals

**Goals:**
- The review box encloses the prompt at any length, wrapping within a terminal-derived width.
- Width tracks the terminal and reflows on resize, consistent with the compose prompt's sizing.

**Non-Goals:**
- No change to `gen.Draft` or to how the prompt is stored/submitted — this is presentation only.
- No scrolling or truncation of very long prompts (wrapping is sufficient).
- No restyling of the box beyond the width fix.

## Decisions

### Set an explicit width on the box, let lipgloss wrap

Give the box style a width derived from `m.width`, mirroring the compose prompt's `max(20, m.width-4)` convention, and account for the border + horizontal padding so the inner text width is correct. With a set `Width`, lipgloss wraps the content to that width and draws a border that encloses it.

- Content/inner width = box width minus the 2 border columns and 2 padding columns. Set the width on the style (lipgloss treats `Width` as the content box and adds border/padding around it), so compute the style width as `max(someMin, m.width - margin)` where `margin` leaves room at the screen edge, matching the compose prompt's `-4`.
- Guard the degenerate case: when `m.width` is 0 (before the first `WindowSizeMsg`), fall back to a sensible minimum width so tests and first paint don't produce a zero-width box.
- Alternative considered: pre-wrapping the string with `lipgloss.NewStyle().Width(w).Render(...)` before bordering, or using `wordwrap`/`wrap` from `muesli/reflow`. Rejected — setting `Width` directly on the bordered style is the idiomatic lipgloss approach and needs no extra dependency.

### Width source

Use the model's stored `m.width` (already maintained by the `WindowSizeMsg` handler). No new state. Resize already triggers a re-render, so the box reflows for free once width is honored.

## Risks / Trade-offs

- [Off-by-one on inner width from border+padding math] → Verify by rendering at a known width in a test and asserting no stripped line exceeds the box width; adjust the margin constant if needed.
- [Zero width before first resize] → Mitigated by a minimum-width fallback when `m.width <= 0`.
