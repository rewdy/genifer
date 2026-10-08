# Spec Delta

## MODIFIED Requirements

### Requirement: Capability-adaptive generate form

The generate form SHALL reflect the selected model's capabilities: controls for parameters the model does not support SHALL be hidden or disabled, and the option to attach reference images SHALL be available only when the model accepts them.

When the selected model accepts reference images, the form SHALL let the user attach one or more images, either from the operating-system clipboard via a dedicated keypress or by entering a local file path on its own line in the prompt prefixed with a reserved sigil. At submit, the form SHALL extract sigil-prefixed path lines, resolve them (including leading `~` expansion), attach each resolved image, and remove those lines from the prompt text sent to the provider. The form SHALL display a hint naming the paste key, SHALL show an indicator of each attached image, and SHALL let the user remove attached images. Reading the clipboard or a referenced file SHALL be best-effort: a clipboard that holds no image, a path that is missing or unreadable, or data that cannot be decoded as a supported image SHALL produce a non-fatal status message, never an application error. Attached images SHALL be carried into the generation request for that model.

#### Scenario: Model supports reference images

- **WHEN** the selected model accepts reference images
- **THEN** the form offers an action to attach one or more reference images and shows a hint naming the key that pastes an image from the clipboard

#### Scenario: Model does not support a parameter

- **WHEN** the selected model does not support a given generation parameter
- **THEN** the form does not present a control for that parameter

#### Scenario: Switching models updates the form

- **WHEN** the user changes the selected model
- **THEN** the form's available controls update to match the new model's capabilities

#### Scenario: Paste action hidden for incapable model

- **WHEN** the selected model does not accept reference images
- **THEN** the form offers no reference-image attach action and no paste hint, and the paste keypress has no effect

#### Scenario: Paste an image from the clipboard

- **WHEN** the model accepts reference images and the user presses the paste key while the clipboard holds a usable image
- **THEN** the image is validated and attached to the draft, and the form shows an indicator for the newly attached image

#### Scenario: Multiple attached images are each indicated

- **WHEN** the user has attached more than one reference image
- **THEN** the form shows a distinct indicator for each attached image together with a visible count

#### Scenario: Remove an attached image

- **WHEN** the user invokes the remove action with at least one image attached
- **THEN** one attached image is removed and the indicator updates to reflect the remaining images, returning to the no-image state when none remain

#### Scenario: Clipboard holds no usable image

- **WHEN** the user presses the paste key and the clipboard holds no image or holds data that cannot be decoded as a supported image
- **THEN** the form shows a transient, non-fatal status message and attaches nothing

#### Scenario: Clipboard cannot be read

- **WHEN** the user presses the paste key but the clipboard is unavailable in the current environment (for example, no display, or over a remote session)
- **THEN** the form reports that no image could be read without erroring the application, and generation remains possible without a reference image

#### Scenario: Attached images are sent with the generation

- **WHEN** the user generates with one or more reference images attached to a reference-capable model
- **THEN** the attached images are included in the generation request for that model

#### Scenario: Attach by file path

- **WHEN** the model accepts reference images and the user submits a prompt containing a line that is a reserved sigil followed by a path to a readable, supported image
- **THEN** the image at that path is validated and attached, the sigil line is removed from the prompt text sent to the provider, and the form shows an indicator for the attached image

#### Scenario: File path is missing or unreadable

- **WHEN** the user submits a prompt whose sigil-prefixed path refers to a file that does not exist or cannot be read
- **THEN** the form shows a transient, non-fatal status message, attaches nothing for that path, and does not send the generation until the issue is addressed

#### Scenario: File path is not a usable image

- **WHEN** the user submits a prompt whose sigil-prefixed path refers to a file that cannot be decoded as a supported image or exceeds the size cap
- **THEN** the form shows a transient, non-fatal status message and attaches nothing for that path
