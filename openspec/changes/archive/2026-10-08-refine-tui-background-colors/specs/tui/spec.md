# Spec Delta

## MODIFIED Requirements

### Requirement: Full-screen application shell

The system SHALL run as a full-screen terminal application with a visually distinct header displayed prominently, and SHALL restore the terminal to its prior state on exit. The application SHALL paint a consistent app background across the entire surface — header, body, and every embedded widget (such as the model list and the prompt input) — so the terminal's own default background never shows through as a different color. Input areas, such as the prompt field and the picker's filter field, SHALL be painted a slightly lighter shade of the app background so they are distinguishable as places to type.

#### Scenario: Launch

- **WHEN** genifer is started in a capable terminal
- **THEN** it takes over the full screen and displays the styled header

#### Scenario: Clean exit

- **WHEN** the user quits genifer
- **THEN** the terminal is restored to its normal state with no residual full-screen artifacts

#### Scenario: Resize

- **WHEN** the terminal is resized
- **THEN** the layout reflows to the new dimensions without corrupting the display

#### Scenario: Uniform background across the surface

- **WHEN** any screen of genifer is displayed in a terminal whose default background differs from the app's
- **THEN** the header, the body content, and every embedded widget are painted with the app background, with no region left in the terminal's default background color

#### Scenario: Short rows fill the full width

- **WHEN** a rendered row of content (for example a list entry, a prompt line, or a status line) is narrower than the available width
- **THEN** the remaining cells of that row are filled with the app background rather than the terminal's default

#### Scenario: Content is inset like the header

- **WHEN** any screen of genifer is displayed
- **THEN** the main content area is inset from the terminal edges by the same horizontal padding as the header, so the content is not butted against the edge

#### Scenario: Input areas are visually distinct

- **WHEN** a screen showing an input area (the prompt field or the picker's filter field) is displayed
- **THEN** that input area is painted a slightly lighter shade of the app background, so it is distinguishable from the surrounding surface
