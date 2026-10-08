# Spec: `just release` recipe

- **Status:** Shipped
- **Date:** 2026-10-08
- **Author:** andrew.meyer (with Kiro)

## Problem

Cutting a version is manual and easy to get wrong: the reported version comes from
`git describe --tags`, so a release *is* a tag — but nothing stops me tagging a dirty
tree, re-using a tag, skipping the build/test/vet gate, or creating a lightweight tag
that `git describe` and GitHub treat poorly. I want a single command that makes "do it
properly" the path of least resistance and refuses to cut a sloppy tag.

## Acceptance criteria

- [ ] `just release v0.3.0` runs the full gate and, on success, creates an annotated tag `v0.3.0` locally.
- [ ] The version argument is required and validated as `vMAJOR.MINOR.PATCH` (optionally a `-prerelease` / `+build` suffix); anything else is rejected with a clear message and no tag is created.
- [ ] The recipe refuses to run when the working tree is dirty (uncommitted or untracked changes), with a message naming the problem.
- [ ] The recipe refuses to run when the tag already exists locally or on the `origin` remote, with a message.
- [ ] The gate runs `go build ./...`, `go test ./...`, and `go vet ./...`; any failure aborts before tagging.
- [ ] The tag is **signed** (`git tag -s`) when a signing key is configured (`user.signingkey` set or `tag.gpgSign=true`), otherwise a plain annotated tag (`git tag -a`). The chosen mode is reported.
- [ ] The annotation message is `v0.3.0 — <summary>`; when no summary is supplied the recipe opens `$EDITOR` for the message rather than writing a bare one.
- [ ] The recipe does **not** push. On success it prints the exact `git push origin v0.3.0` command to run next.
- [ ] `just bump patch|minor|major` computes the next version from the latest reachable tag and routes through `release` (same guards, gate, signing, no-push). The level argument defaults to `patch`, so bare `just bump` is a patch bump.
- [ ] `bump` rejects an unknown level (anything other than `patch`/`minor`/`major`) with a clear message.
- [ ] `bump` bases the computation on `git describe --tags --abbrev=0`; a pre-release/build suffix on the base tag is dropped and the numeric component incremented (standard SemVer bump).
- [ ] When no tag exists yet, `bump` does not guess: it reports there is no base version and tells the user to cut the first release explicitly (e.g. `just release v0.1.0`).
- [ ] An optional summary passes through: `just bump minor "add onboarding"` tags with that annotation message.
- [ ] A `just --list` (or `just`) shows the recipes with one-line descriptions.
- [ ] README documents the release and bump flow.

## Approach

Add a `justfile` at the repo root with a `release` recipe written as a `#!/usr/bin/env bash`
shebang recipe (one shell for the whole script, `set -euo pipefail`) so the guards read as a
coherent sequence. Recipe takes a positional `version` parameter and an optional `summary`
(`release version summary=""`).

Guard order (fail fast, cheapest/most-likely-wrong first):
1. Validate `version` against a SemVer-with-`v` regex.
2. Clean-tree check: `git status --porcelain` must be empty.
3. Tag-exists check: `git rev-parse -q --verify "refs/tags/$version"` (local) and
   `git ls-remote --tags origin "$version"` (remote).
4. Gate: `go build ./...` → `go test ./...` → `go vet ./...`.
5. Signing detection: `-s` if `git config user.signingkey` or `tag.gpgSign` is set, else `-a`.
6. Create the tag; print the push command.

Keep the Go commands as the single source of truth by matching what `AGENTS.md` already prescribes.
A couple of other convenience recipes (`build`, `test`, `vet`, maybe a `check` that runs all three)
are in scope only insofar as they let `release` reuse them; the release flow is the point.

A `bump level="patch"` recipe computes the next version and then **routes through `release`** by
invoking `just release <computed> <summary>` — so `release` stays the single owner of the guards,
gate, signing, and no-push behavior, and `bump` only does the arithmetic:
1. Reject a level other than `patch|minor|major`.
2. Base = `git describe --tags --abbrev=0`; if that fails (no tags), stop and tell the user to cut
   the first release explicitly.
3. Strip any `-prerelease`/`+build` suffix, parse `vMAJOR.MINOR.PATCH`, increment the chosen
   component (minor/major zero the lower components), and hand the result to `release`.

No ADR here — this is a localized tooling choice (`just` over `make`), not a constraint that reaches
beyond this file.

## Out of scope

- Pushing the tag or any remote mutation — the recipe stops at a local tag and prints the push command.
- Drafting a GitHub Release (`gh release create`) or generating release notes/changelog.
- A CI workflow that triggers on tag push.
- Bumping any version constant or editing files as part of the release (there is none to bump).
- Porting the existing build/test/vet commands away from their documented form, or replacing the committed binary workflow.
- Windows / non-bash environments: the recipe assumes a bash + git + Go toolchain on macOS/Linux.

## Open questions

None outstanding — decisions settled during discussion (require validated `v` prefix; auto-detect
signing; create locally and print the push command; stop at the tag; `bump` is a separate recipe
that routes through `release`; no base tag means bump refuses and asks for an explicit first
release; pre-release suffix on the base is dropped on bump).
