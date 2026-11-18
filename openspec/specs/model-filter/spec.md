# model-filter Specification

## Purpose

Lets users narrow the model picker by advertised capabilities such as transparent backgrounds, animated GIF, reference images, seeds, and vector output, instead of relying only on typed name search.

## Requirements

### Requirement: Model picker offers capability filters

In addition to text search, the model picker SHALL offer toggleable capability filters for transparent backgrounds, animated GIF output, reference-image input, seed support, and vector/SVG output. Active filters SHALL be visibly indicated, SHALL be clearable in one action, and SHALL be discoverable from the picker's key hints. The filters SHALL be available whenever the model list is ready and SHALL NOT block any existing picker action.

#### Scenario: Capability filters are available

- **WHEN** the model list is ready
- **THEN** the picker offers filters for transparency, animated GIF, reference images, seed, and vector/SVG output

#### Scenario: Active filters are visible and clearable

- **WHEN** one or more capability filters are active
- **THEN** the picker indicates which filters are active and offers a single action that clears all of them

#### Scenario: Filter controls are discoverable

- **WHEN** the picker is displayed
- **THEN** key hints name how to open and clear capability filters

### Requirement: Capability filters narrow results and compose with text search

A model SHALL be visible only when it satisfies every active capability filter. A typed text query SHALL compose with capability filters so both constraints apply at once. Capability filtering SHALL preserve the picker's favorites-first ordering and relative model order, SHALL keep the current selection valid where possible, and SHALL communicate an empty result when no model matches. Clearing the filters SHALL restore the unfiltered list.

#### Scenario: Single capability filter

- **WHEN** only the animated GIF filter is active
- **THEN** the picker shows only models that support animated GIF

#### Scenario: Multiple capability filters combine

- **WHEN** the transparency and seed filters are both active
- **THEN** the picker shows only models that support both transparency and seeds

#### Scenario: Text query composes with capability filters

- **WHEN** a text query and a capability filter are both active
- **THEN** the picker shows only models matching the text query that also satisfy the capability filter

#### Scenario: Favorites ordering is preserved

- **WHEN** capability filters are active and matching models include favorites
- **THEN** matching favorites appear before other matching models in their existing relative order

#### Scenario: No model matches

- **WHEN** active filters match no available model
- **THEN** the picker shows an empty-result message rather than a broken or misleading list

#### Scenario: Clearing restores the list

- **WHEN** the user clears all active capability filters
- **THEN** the picker shows all models subject only to any remaining text query

### Requirement: Capability filters use advertised model capabilities

Capability filters SHALL be evaluated solely from the capabilities each model reports through the provider interface. A capability that a model does not advertise SHALL be treated as unsupported. No filter SHALL depend on hard-coded model identifiers, so models that newly advertise a capability match the corresponding filter without a code change.

#### Scenario: Unknown capability is excluded

- **WHEN** a model does not report support for a capability
- **THEN** that model is excluded when the corresponding filter is active

#### Scenario: New capable model matches automatically

- **WHEN** discovery starts reporting a capability for a model that previously lacked it
- **THEN** that model matches the corresponding filter on the next refresh without a code change

#### Scenario: Vector models match the vector filter

- **WHEN** a model advertises vector/SVG output
- **THEN** it matches the vector/SVG capability filter
