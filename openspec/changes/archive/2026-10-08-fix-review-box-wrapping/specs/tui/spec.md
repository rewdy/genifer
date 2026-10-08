# Spec Delta

## ADDED Requirements

### Requirement: Review step frames the prompt within a bounded box

At the review step, the system SHALL display the prompt inside a bordered box whose width is bounded by the current terminal width. Prompt text that is longer than the available width SHALL wrap within the box, and the box border SHALL fully enclose the wrapped text — no prompt text SHALL extend past the border, and the top and bottom edges SHALL align with the sides regardless of prompt length. The box SHALL reflow to the terminal width when the terminal is resized.

#### Scenario: Short prompt

- **WHEN** the review step is shown for a prompt that fits on one line
- **THEN** the prompt appears inside a bordered box whose border fully encloses it

#### Scenario: Long prompt wraps inside the box

- **WHEN** the review step is shown for a prompt longer than the available width
- **THEN** the prompt wraps onto multiple lines inside the box and no line extends past the right border

#### Scenario: Box reflows on resize

- **WHEN** the terminal is resized while the review step is shown
- **THEN** the box width adjusts to the new terminal width and the prompt re-wraps to fit, with the border still fully enclosing the text
