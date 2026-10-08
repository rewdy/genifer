# genifer demo tape — render with:  vhs demo/genifer-demo.tape
#
# Produces demo/genifer.gif. Run from a shell where OPENROUTER_API_KEY is
# exported; VHS inherits your environment, so the key is never stored here.
#
# Prerequisite: `genifer` must be on your PATH (e.g. `go install .` or
# `go build -o /usr/local/bin/genifer .`). This tape only records the demo.
#
# Deterministic and spends no credit: it walks the model picker, filters,
# selects a model, enters a prompt, and stops at the review screen — it does
# NOT press Enter to generate.

Output demo/genifer.gif

Set Shell zsh
Set FontSize 18
Set Width 1200
Set Height 760
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 70ms

# --- Launch genifer ---------------------------------------------------------
Type "genifer"
Sleep 500ms
Enter
Sleep 4s

# --- Browse the model list --------------------------------------------------
Down
Sleep 600ms
Down
Sleep 600ms
Down
Sleep 800ms

# --- Filter the list --------------------------------------------------------
Type "/"
Sleep 500ms
Type "flux"
Sleep 1500ms
Escape
Sleep 800ms

# --- Pick a model -----------------------------------------------------------
Enter
Sleep 1500ms

# --- Compose a prompt -------------------------------------------------------
Type "a cute robot painting a sunset over the ocean, retro poster style"
Sleep 1500ms

# --- Review the prompt (stops here — no generation) -------------------------
Ctrl+S
Sleep 3s
