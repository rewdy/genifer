package config

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Value is an unresolved configuration string. It may be a literal, an
// environment-variable reference of the form "{env:NAME}", or a command
// reference of the form "{cmd:...}". Resolution is single-level: a resolved
// value is never scanned again for directives.
type Value string

// Resolve evaluates v into its concrete string.
//
//   - literal:      returned verbatim
//   - {env:NAME}:   the value of environment variable NAME (error if unset)
//   - {cmd:...}:    the trimmed stdout of running "..." via the shell
//     (error if the command cannot run or exits non-zero)
//
// Commands run only when Resolve is called, so a command-backed value that is
// never needed is never executed (lazy evaluation).
func (v Value) Resolve(ctx context.Context) (string, error) {
	raw := string(v)
	name, inner, ok := directive(raw)
	if !ok {
		return raw, nil
	}
	switch name {
	case "env":
		val, present := os.LookupEnv(inner)
		if !present {
			return "", fmt.Errorf("config: environment variable %q is not set", inner)
		}
		return val, nil
	case "cmd":
		return runCommand(ctx, inner)
	default:
		// Unknown directive: treat the whole string as a literal.
		return raw, nil
	}
}

// IsCommand reports whether resolving v would execute an external command.
// Useful for deciding whether a value is safe to resolve eagerly.
func (v Value) IsCommand() bool {
	name, _, ok := directive(string(v))
	return ok && name == "cmd"
}

// directive parses a "{name:inner}" wrapper. It returns ok=false for any string
// that is not exactly one such wrapper, so literal text containing braces is
// left untouched.
func directive(s string) (name, inner string, ok bool) {
	if len(s) < 3 || s[0] != '{' || s[len(s)-1] != '}' {
		return "", "", false
	}
	body := s[1 : len(s)-1]
	colon := strings.IndexByte(body, ':')
	if colon <= 0 {
		return "", "", false
	}
	name = body[:colon]
	inner = body[colon+1:]
	if strings.ContainsAny(name, "{} ") {
		return "", "", false
	}
	return name, inner, true
}

func runCommand(ctx context.Context, command string) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", fmt.Errorf("config: empty command directive")
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(ee.Stderr))
			if stderr != "" {
				return "", fmt.Errorf("config: command %q failed: %v: %s", command, err, stderr)
			}
		}
		return "", fmt.Errorf("config: command %q failed: %w", command, err)
	}
	return strings.TrimSpace(string(out)), nil
}
