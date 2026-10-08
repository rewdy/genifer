# Tasks

## 1. Implement tilde expansion

- [x] 1.1 In `internal/config/outputdir.go`, add a `~` / `~/...` expansion case to `ResolveOutputDir` (bare `~` → home, `~/x` → `<home>/x`), resolving home via an injectable/`os.UserHomeDir`-backed lookup so it stays testable and does not import the `tui` package. Verify with `go build ./...`.
- [x] 1.2 Update the `ResolveOutputDir` doc comment to list the `~` case alongside unset / `.` / `pwd` / relative / absolute. Verify the comment matches the implemented cases.
- [x] 1.3 Add table cases to `internal/config/outputdir_test.go` for `~` and `~/Downloads/genifer` (asserting expansion to the fake home), and verify `go test ./internal/config/` passes.

## 2. Verify end to end

- [x] 2.1 Run `go test ./...` and `go vet ./...` and confirm both pass.
