// Package tui implements the full-screen Bubble Tea application shell: header,
// model picker, capability-adaptive generate form, status, and keybindings.
//
// For reference-capable models the compose form attaches reference images two
// ways: a clipboard paste keypress (read out-of-band via the imagePaster seam)
// and sigil-prefixed file-path lines in the prompt, resolved at submit. Both
// paths validate the image, surface it as an inline pill, and feed
// gen.Draft.ReferenceImages. Clipboard and file access are best-effort and
// never fatal.
package tui
