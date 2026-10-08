# genifer task runner. Run `just` or `just --list` to see available recipes.

# Show the available recipes.
default:
    @just --list

# Build everything.
build:
    go build ./...

# Run all tests.
test:
    go test ./...

# Vet all packages.
vet:
    go vet ./...

# Run the full gate: build, test, vet.
check: build test vet

# Cut a release: validate, gate, and create an annotated (signed when possible)
# tag locally. Does NOT push — prints the push command on success.
#   just release v0.3.0
#   just release v0.3.0 "config command and first-run onboarding"
release version summary="":
    #!/usr/bin/env bash
    set -euo pipefail

    # Colors — disabled when NO_COLOR is set or output isn't a terminal.
    if [[ -z "${NO_COLOR:-}" && -t 1 && -t 2 ]]; then
        red=$'\033[31m'; green=$'\033[32m'; yellow=$'\033[33m'
        cyan=$'\033[36m'; bold=$'\033[1m'; dim=$'\033[2m'; reset=$'\033[0m'
    else
        red=; green=; yellow=; cyan=; bold=; dim=; reset=
    fi

    version="{{ version }}"
    summary="{{ summary }}"

    # 1. Validate the version: vMAJOR.MINOR.PATCH, optional -prerelease / +build.
    if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.-]+)*$ ]]; then
        echo "${red}release: invalid version '$version'${reset}" >&2
        echo "${dim}         expected SemVer with a leading v, e.g. v0.3.0 or v1.2.3-rc.1${reset}" >&2
        exit 1
    fi

    # 2. Refuse a dirty working tree (uncommitted or untracked changes).
    if [[ -n "$(git status --porcelain)" ]]; then
        echo "${red}release: working tree is dirty — commit or stash before tagging:${reset}" >&2
        git status --short >&2
        exit 1
    fi

    # 3. Refuse an already-used tag, locally or on origin.
    if git rev-parse -q --verify "refs/tags/$version" >/dev/null; then
        echo "${red}release: tag '$version' already exists locally${reset}" >&2
        exit 1
    fi
    if [[ -n "$(git ls-remote --tags origin "$version" 2>/dev/null)" ]]; then
        echo "${red}release: tag '$version' already exists on origin${reset}" >&2
        exit 1
    fi

    # 4. Gate: build, test, vet. Any failure aborts before tagging.
    echo "${cyan}release: running build/test/vet gate...${reset}"
    go build ./...
    go test ./...
    go vet ./...

    # 5. Detect signing: -s when a key/config is present, else plain annotated -a.
    if [[ -n "$(git config --get user.signingkey || true)" || "$(git config --get tag.gpgSign || true)" == "true" ]]; then
        sign_flag="-s"
        sign_desc="signed"
    else
        sign_flag="-a"
        sign_desc="annotated (unsigned)"
    fi

    # 6. Create the tag. With a summary, write the one-line message; without one,
    #    open the editor so the message is deliberate rather than bare.
    echo "${cyan}release: creating $sign_desc tag ${bold}$version${reset}"
    if [[ -n "$summary" ]]; then
        git tag "$sign_flag" "$version" -m "$version — $summary"
    else
        git tag "$sign_flag" "$version"
    fi

    echo ""
    echo "${green}✓ Created $sign_desc tag ${bold}$version${reset}"
    echo "${dim}  Push it when ready:${reset}"
    echo ""
    echo "    ${yellow}git push origin $version${reset}"

# Bump the latest tag and release the next version. Routes through `release`,
# so all the same guards, gate, and signing apply. Level defaults to patch.
#   just bump            # patch: v0.3.0 -> v0.3.1
#   just bump minor      # v0.3.1 -> v0.4.0
#   just bump major "big rewrite"
bump level="patch" summary="":
    #!/usr/bin/env bash
    set -euo pipefail

    # Colors — disabled when NO_COLOR is set or output isn't a terminal.
    if [[ -z "${NO_COLOR:-}" && -t 1 && -t 2 ]]; then
        red=$'\033[31m'; cyan=$'\033[36m'; bold=$'\033[1m'; dim=$'\033[2m'; reset=$'\033[0m'
    else
        red=; cyan=; bold=; dim=; reset=
    fi

    level="{{ level }}"
    summary="{{ summary }}"

    case "$level" in
        patch|minor|major) ;;
        *)
            echo "${red}bump: unknown level '$level' — expected patch, minor, or major${reset}" >&2
            exit 1
            ;;
    esac

    # Base version = latest reachable tag. No tags yet => don't guess.
    if ! base="$(git describe --tags --abbrev=0 2>/dev/null)"; then
        echo "${red}bump: no existing tag to bump from${reset}" >&2
        echo "${dim}      cut the first release explicitly, e.g. just release v0.1.0${reset}" >&2
        exit 1
    fi

    # Strip any -prerelease / +build suffix, then parse vMAJOR.MINOR.PATCH.
    core="${base%%[-+]*}"
    if [[ ! "$core" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
        echo "${red}bump: latest tag '$base' is not a vMAJOR.MINOR.PATCH version${reset}" >&2
        echo "${dim}      bump from a SemVer tag, or cut the next one with just release vX.Y.Z${reset}" >&2
        exit 1
    fi
    major="${BASH_REMATCH[1]}"
    minor="${BASH_REMATCH[2]}"
    patch="${BASH_REMATCH[3]}"

    case "$level" in
        patch) patch=$((patch + 1)) ;;
        minor) minor=$((minor + 1)); patch=0 ;;
        major) major=$((major + 1)); minor=0; patch=0 ;;
    esac
    next="v${major}.${minor}.${patch}"

    echo "${cyan}bump: $base -> ${bold}$next${reset}${cyan} ($level)${reset}"
    just release "$next" "$summary"
