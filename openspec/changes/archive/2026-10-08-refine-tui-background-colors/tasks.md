# Tasks

## 1. Central style source

- [x] 1.1 Add `internal/tui/styles.go` with the shared app background color and prebuilt widget styles (textarea focused/blurred, list styles, row-fill helper), and verify `go build ./...` succeeds
- [x] 1.2 Point `header.go` and `delegate.go` at the shared background constant so the color is defined once, and verify `go test ./internal/tui/...` still passes

## 2. Widget background coverage

- [x] 2.1 Apply the app background to the prompt textarea's focused and blurred styles in `New` (model.go), replacing the default `CursorLine` black band, and verify a `tui` test asserts the rendered textarea emits the app background rather than `0`
- [x] 2.2 Apply the app background to the picker list's styles (filter prompt, pagination dots, no-items, status) in `New`, and verify a `tui` test renders the picker with a filter/pagination line and finds no unstyled background
- [x] 2.3 Fill each `modelDelegate` row to the list width with the app background instead of only truncating, and verify a delegate test shows a short row padded to full width with the background color

## 3. Whole-surface backstop

- [x] 3.1 Add a line-padding helper that fills each composed line to the terminal width with the app background, and apply it in `Model.View`, and verify a `tui` test renders a view at a known width and asserts every line measures exactly that width
- [x] 3.2 Verify the body, picker, compose, aspect, review, result, and onboarding screens all render every line at full width with the app background via a table-driven `tui` test over the phases

## 4. Integration verification

- [x] 4.1 Run `just check` (build + test + vet) and verify it passes
- [ ] 4.2 Manually run `go run .` in a terminal whose default background differs from `#1e1c32` and verify no terminal-default background shows through on any screen

## Workflow follow-up

- Archive the change after the project's review requirements are satisfied.
