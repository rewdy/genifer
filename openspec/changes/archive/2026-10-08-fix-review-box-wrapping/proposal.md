# Proposal

## Why

On the Review screen, a prompt longer than one line breaks the bordered box: the border is sized to the terminal rather than wrapped to fit, so wrapped prompt text overflows past the right border and the top/bottom edges no longer line up with the sides (see the attached screenshot). The review box should always frame the prompt cleanly regardless of prompt length.

## What Changes

- The Review screen's prompt box SHALL wrap long prompt text within a bounded width and render a border that fully encloses the wrapped text, so multi-line prompts no longer overflow the box.
- The box width SHALL be derived from the current terminal width (consistent with how the compose prompt is already sized) so the box reflows on resize instead of using an unbounded natural width.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `tui`: the "Status and keybindings" / review-step rendering behavior is clarified so the review prompt box constrains and wraps the prompt. (Delta targets the existing `tui` capability; a review-rendering requirement is added describing the enclosed, wrapped box.)

## Impact

- `internal/tui/view.go`: `reviewView` sets an explicit, terminal-derived width on the bordered box so lipgloss wraps the prompt inside the border.
- `internal/tui/generate_test.go` (or a sibling): a test rendering a multi-line/long prompt asserts the review box encloses the text without overflow.
- No provider, config, or dependency changes.
