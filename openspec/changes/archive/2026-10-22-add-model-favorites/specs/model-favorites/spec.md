# Spec Delta

## Purpose

Lets users mark models as favorites so their preferred models appear at the top
of the model picker, with the favorite set remembered across sessions in
app-owned state rather than in the hand-edited configuration file.

## ADDED Requirements

### Requirement: Users can toggle a model as a favorite

The model picker SHALL let the user mark the highlighted model as a favorite and
unmark a model that is already favorited, using a single keypress available
whenever the picker is focused and its filter input is not active. Toggling SHALL
take effect immediately for the current session, SHALL NOT open a dialog or
require confirmation, and SHALL leave the highlighted model as the highlighted
row so a user can toggle several models in sequence.

The picker SHALL render a yellow star (`★`, U+2605) immediately before the name
of every favorited model so the current state of each row is apparent. The star
SHALL be visually distinct from the model name and from the muted pricing and
capability detail, and it SHALL NOT replace or obscure any of them. The picker
SHALL surface a discoverable hint naming the toggle key.

#### Scenario: Mark a model as a favorite

- **WHEN** the picker is focused on a model that is not favorited and the user presses the toggle key
- **THEN** that model becomes a favorite, its row shows a yellow star next to the name, and it moves to the top of the list

#### Scenario: Unmark a favorite

- **WHEN** the picker is focused on a model that is favorited and the user presses the toggle key
- **THEN** that model stops being a favorite, its yellow star is removed, and it returns to its normal position in the list

#### Scenario: Toggle is a single keypress

- **WHEN** the user presses the toggle key on a model
- **THEN** the favorite state changes immediately with no confirmation step or dialog

#### Scenario: Selection stays on the toggled model

- **WHEN** the user toggles a model that moves to a new position in the list
- **THEN** the highlight remains on that same model after the reorder

#### Scenario: Toggle key is inactive while filtering

- **WHEN** the picker's filter input is active and the user types the toggle key's character
- **THEN** the character is entered into the filter query and no favorite changes

#### Scenario: Favorited row shows a yellow star

- **WHEN** a favorited model is rendered in the picker
- **THEN** its row shows a yellow `★` before the name, in addition to the model name and any pricing or capability detail, each still readable

#### Scenario: Non-favorited rows show no star

- **WHEN** a model that is not favorited is rendered in the picker
- **THEN** its row contains no star, so favorited and non-favorited rows are distinguishable at a glance

### Requirement: Favorited models appear at the top of the picker

The picker SHALL order favorited models before all non-favorited models. Within
the favorited group, and within the non-favorited group, models SHALL keep the
provider's existing relative order. When a filter query is active, the visible
results SHALL follow the same rule: matching favorites first, then other matches
in provider order. A reorder caused by toggling SHALL NOT change which model is
selected for generation.

When no models are favorited, the picker SHALL show the unmodified provider
order.

#### Scenario: Favorites sort first

- **WHEN** the picker displays models and at least one model is favorited
- **THEN** all favorited models appear above the non-favorited models

#### Scenario: Relative order is preserved within groups

- **WHEN** several models are favorited
- **THEN** those favorites appear in the same relative order the provider listed them, and the non-favorited models keep their own relative order below

#### Scenario: No favorites leaves the list unchanged

- **WHEN** no models are favorited
- **THEN** the picker shows the provider's model order

#### Scenario: Filtering keeps favorites first

- **WHEN** a filter query matches both favorited and non-favorited models
- **THEN** the matching favorites appear above the other matches, each group keeping provider order

#### Scenario: Reordering does not change the chosen model

- **WHEN** toggling a favorite reorders the list
- **THEN** the model that would be used for generation remains the one the user highlighted before the toggle

### Requirement: Favorites persist in app-owned state

The system SHALL persist the set of favorited model identifiers in the app-owned
`state.json` in the config directory, alongside the other remembered selections.
It SHALL NOT write favorites to `config.yaml`, which remains user-owned. Saving a
favorite SHALL preserve the other persisted selections, and saving another
selection SHALL preserve the favorite set. When `state.json` is missing,
unparsable, or does not contain favorites, the app SHALL start with an empty
favorite set and continue normally.

#### Scenario: Favorites survive restart

- **WHEN** the user favorites a model, quits, and relaunches genifer
- **THEN** that model is still a favorite and still appears at the top of the picker

#### Scenario: Favorites are not written to config.yaml

- **WHEN** the user toggles a favorite
- **THEN** `state.json` is updated and `config.yaml` is left byte-for-byte unchanged

#### Scenario: Persisted selections do not clobber each other

- **WHEN** the user favorites a model and then selects a different model for generation
- **THEN** the favorite set is retained in `state.json` along with the newly remembered last-used model

#### Scenario: Missing or invalid state is harmless

- **WHEN** `state.json` is absent or cannot be parsed
- **THEN** genifer starts with no favorites and no error, and recreates the file on the next write

### Requirement: Favorites that no longer exist are inert

A favorited model identifier that is not present in the provider's current model
list SHALL be ignored for display and ordering: it SHALL NOT create an empty row,
SHALL NOT error, and SHALL NOT prevent the rest of the list from rendering. The
system SHALL retain the stored identifier rather than silently deleting it, so a
model that reappears later is favorited again. The picker's status or help area
SHALL NOT present stale favorites as an error condition.

#### Scenario: Stale favorite is not displayed

- **WHEN** `state.json` contains a favorite whose model the provider no longer lists
- **THEN** the picker shows only the models the provider returns, with no empty row and no error

#### Scenario: Stale favorite is retained in storage

- **WHEN** a stored favorite is absent from the current model list
- **THEN** its identifier remains in `state.json` after subsequent favorite or selection writes

#### Scenario: Favorite reappears

- **WHEN** a previously favorited model identifier is present in the provider list again
- **THEN** that model is displayed as a favorite at the top of the picker without further user action
