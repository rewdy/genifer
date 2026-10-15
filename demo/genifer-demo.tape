# genifer demo tape — render with:  vhs demo/genifer-demo.tape
#
# Produces demo/genifer.gif. Run from a shell where OPENROUTER_API_KEY is
# exported; VHS inherits your environment, so the key is never stored here.
#
# Prerequisite: `genifer` must be on your PATH (e.g. `go install .` or
# `go build -o /usr/local/bin/genifer .`). This tape only records the demo.
#
# Generates a real image: it walks the model picker, filters to Nano Banana 2
# Lite (Google's fastest image model), enters a prompt, generates, and shows
# the final result screen. Spends a small amount of credit per run.

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
# Enter (not Escape) accepts the filter: Escape cancels it and restores the
# full list, which would leave an arbitrary model selected.
Type "/"
Sleep 500ms
Type "nano banana 2 lite"
Sleep 1500ms
Enter
Sleep 800ms

# --- Pick the filtered model ------------------------------------------------
Enter
Sleep 1500ms

# --- Compose a prompt -------------------------------------------------------
Type "a cute robot painting a sunset over the ocean, retro poster style"
Sleep 1500ms

# --- Review the prompt ------------------------------------------------------
Ctrl+S
Sleep 2s

# --- Generate and show the final result screen ------------------------------
Enter
Sleep 12s
