# Tasks

## 1. Bound and wrap the review box

- [x] 1.1 In `reviewView` (`internal/tui/view.go`), set an explicit width on the bordered box style derived from `m.width` (mirror the compose prompt's `max(20, m.width-4)` convention), with a minimum-width fallback when `m.width <= 0`, so lipgloss wraps the prompt inside the border. Verify `go build ./...` succeeds and manual run shows a long prompt enclosed.
- [x] 1.2 Add a test (in `internal/tui/generate_test.go` or a sibling) that renders `reviewView` at a known `m.width` with a long/multi-line prompt, strips ANSI, and asserts no rendered line exceeds the box width and that the top/bottom border rows match the side width (box fully encloses the text). Verify `go test ./internal/tui/` passes.
- [x] 1.3 Confirm the existing `TestReviewViewShowsCount` still passes (set `m.width` in that test if the box now depends on it). Verify `go test ./internal/tui/` passes.

## 2. Integration check

- [x] 2.1 Run `go build ./...`, `go test ./...`, and `go vet ./...` (or `just check`) and confirm all pass.
