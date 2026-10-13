// Package tui implements the full-screen Bubble Tea application shell: header,
// model picker, capability-adaptive generate form, prompt history, an in-tool
// settings editor, status, and keybindings.
//
// For reference-capable models the compose form attaches reference images two
// ways: a clipboard paste keypress (read out-of-band via the imagePaster seam)
// and sigil-prefixed file-path lines in the prompt, resolved at submit. Both
// paths validate the image, surface it as an inline pill, and feed
// gen.Draft.ReferenceImages. Clipboard and file access are best-effort and
// never fatal.
//
// A successful generation is appended to the app-owned history store (best-
// effort, gated by the history config option). The history view loads only the
// most-recent entries for responsiveness and points the user to the on-disk
// file for the full record; selecting an entry pre-fills compose, reapplying
// aspect ratio and seed where the active model supports them.
package tui
