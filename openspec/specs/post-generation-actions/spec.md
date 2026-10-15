# post-generation-actions Specification

## Purpose

Defines the actions a user can take on the result screen after a generation
completes, and precisely which composed parameters each action preserves or
clears, so iteration (same prompt / new model, fresh start, or exact re-run) is
direct and predictable.

## Requirements

### Requirement: Result screen offers distinct post-generation actions on success

After a generation succeeds, the result screen SHALL offer, in addition to
opening the saved image, opening its containing folder, and quitting, four distinct actions: edit prompt, try again with a
new model, start a new image, and re-generate. Each action's key hint SHALL be
shown on the result screen.

#### Scenario: Successful generation shows all post-generation actions
- **WHEN** a generation completes successfully
- **THEN** the result screen presents controls for open, open-folder, edit-prompt, try-again-with-new-model, new-image, re-generate, and quit

#### Scenario: Failed generation does not show success-only actions
- **WHEN** a generation fails
- **THEN** edit-prompt, try-again-with-new-model, new-image, and re-generate are not offered, and the existing retry (for retryable failures) and new/quit behavior is unchanged

### Requirement: Try again with a new model preserves composed parameters

The try-again-with-a-new-model action, bound to m, SHALL return the user to the model picker
while preserving the entered prompt text, the selected aspect ratio, and any
attached reference images, so that selecting a model proceeds to a prompt screen
pre-populated with those values.

#### Scenario: Prompt and parameters survive model reselection
- **WHEN** the user chooses try-again-with-a-new-model on the result screen
- **THEN** the model picker is shown, and after selecting a model the prompt text, aspect ratio (where offered by the new model), and reference images from the previous generation are still present

#### Scenario: Reference images dropped when new model cannot accept them
- **WHEN** reference images were attached and the newly selected model does not accept reference images
- **THEN** the reference images are not sent, and the user is informed they were dropped

### Requirement: New image clears composed parameters

The new-image action SHALL return the user to the prompt screen with the prompt
text and reference images cleared, while keeping the currently selected model.
No seed SHALL be carried over from the previous generation; for a seed-capable
model the next generation mints a fresh seed per the seed-control capability.

#### Scenario: New image starts from an empty prompt
- **WHEN** the user chooses new-image on the result screen
- **THEN** the prompt screen is shown with an empty prompt and no reference images attached, and no seed from the previous generation is carried over

### Requirement: Re-generate repeats the completed generation

The re-generate action SHALL immediately start a new generation using the same
model, prompt, aspect ratio, and reference images as the generation that just
completed, without returning the user to the prompt or picker first. For a
seed-capable model, the seed is governed by the seed-control capability
(reuse the same seed to reproduce, or use a new seed to vary); for a model
without seed support, the request repeats as sent.

#### Scenario: Re-generate reuses prompt and parameters
- **WHEN** the user chooses re-generate on the result screen
- **THEN** a new generation begins with the same model, prompt, aspect ratio, and reference images as the just-completed generation

#### Scenario: Re-generate seed follows the seed-control capability
- **WHEN** the user chooses re-generate for a seed-capable model
- **THEN** the seed sent is determined by the user's reuse-or-vary choice as defined by the seed-control capability

### Requirement: Open containing folder

After a successful generation, the result screen SHALL offer an open-folder action that opens the saved image's containing directory in the operating system's default file manager. The image-viewer command override SHALL NOT apply to this action. Opening the folder SHALL preserve the saved result and keep the result screen available. A launch failure SHALL be reported without quitting the application.

#### Scenario: Open saved image folder
- **WHEN** the user presses f on a successful result
- **THEN** the containing directory is opened in the system file manager and the result remains available

#### Scenario: Generation failed
- **WHEN** generation has failed
- **THEN** no open-folder action is offered or run

#### Scenario: File manager launch fails
- **WHEN** the file manager cannot be launched
- **THEN** a non-fatal status message describes the failure

### Requirement: Edit prompt preserves composed parameters

The edit-prompt action, bound to p on a successful result, SHALL return to prompt composition with the current prompt, selected model, aspect ratio, and reference images preserved. The prompt SHALL be focused for editing, and no generation SHALL start until the user reviews and submits the edited draft.

#### Scenario: Edit successful prompt
- **WHEN** the user presses p after a successful generation
- **THEN** the current draft is shown in the focused prompt editor with its model, aspect ratio, and references preserved and no generation starts

### Requirement: Optional native result thumbnail

After an image is successfully saved, the success screen SHALL asynchronously display a native image thumbnail above the saved-path, cost, and seed details when the terminal supports the configured native rendering path and has usable pixel geometry. Initial supported direct sessions SHALL include Ghostty and iTerm2. PNG, JPEG, and WebP outputs SHALL be eligible. SVG outputs SHALL use the normal success screen without a thumbnail. Unsupported terminals, non-terminal output, multiplexer sessions, SSH sessions, unsupported image content, and preview preparation failures SHALL retain the normal success screen without a thumbnail, reserved blank area, character-block approximation, or preview error banner. Saving, recorded cost, history, auto-open, and existing result actions SHALL remain independent of preview success.

#### Scenario: Preview on a supported terminal
- **WHEN** a saved PNG, JPEG, or WebP result is displayed in a supported direct terminal session with usable geometry
- **THEN** its thumbnail appears above the saved-result details without delaying access to result actions

#### Scenario: SVG result
- **WHEN** a saved SVG result is displayed
- **THEN** it uses the normal success screen without a thumbnail and the original SVG remains available through existing open and folder actions

#### Scenario: No preview support
- **WHEN** terminal support or geometry is unavailable, output is redirected, or the session uses tmux, screen, or SSH
- **THEN** the screen remains the normal success screen without thumbnail space, graphics payloads, or a preview warning

#### Scenario: Preview cannot be prepared
- **WHEN** the saved file is unreadable, invalid, unsupported, or exceeds the preview preparation limits
- **THEN** the successful result and its actions remain available without a thumbnail or application error

#### Scenario: Generation failed
- **WHEN** generation fails or is cancelled
- **THEN** no thumbnail preparation or display is started for that outcome

### Requirement: Proportional thumbnail layout

The thumbnail SHALL target a displayed height of 150 terminal pixels, preserving the source aspect ratio and fitting within the result content width and remaining vertical space. It SHALL scale down proportionally when either constraint would otherwise be exceeded, SHALL NOT crop or stretch the image, and SHALL NOT displace the header, saved-result details, status, or result shortcuts. If no usable preview area remains, it SHALL be omitted. Resizing SHALL recompute the layout using current geometry.

#### Scenario: Normal result layout
- **WHEN** a thumbnail at 150 pixels high fits the available space
- **THEN** it is shown at that height with proportional width, subject to terminal cell rounding

#### Scenario: Wide image or short terminal
- **WHEN** the preferred thumbnail would exceed the available content width or remaining height
- **THEN** it is proportionally reduced, or omitted if no usable area remains, while result details and shortcuts remain available

#### Scenario: Resize
- **WHEN** the terminal is resized on the success screen
- **THEN** the prior thumbnail placement is cleared and the preview is refitted or omitted using the updated geometry

### Requirement: Preview lifecycle follows the active result

A thumbnail SHALL belong only to its current successful result. Leaving the result screen, starting another generation, or quitting SHALL clear its native image placement. Background preparation for an obsolete result or terminal geometry SHALL NOT alter the current screen. Ordinary redraws SHALL preserve correct placement without graphics corrupting text or the input stream.

#### Scenario: Navigate away
- **WHEN** the user edits the prompt, chooses another model, starts a new image, re-generates, or quits
- **THEN** the previous preview is removed and does not appear on the next screen

#### Scenario: Obsolete preview completes
- **WHEN** background preview work finishes after its result or geometry has changed
- **THEN** its completion is discarded without changing the current preview or status

#### Scenario: Result actions redraw
- **WHEN** opening the saved file or folder changes status on the success screen
- **THEN** the thumbnail stays correctly placed and all text and shortcuts remain readable
