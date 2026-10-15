# Spec Delta

## ADDED Requirements

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
