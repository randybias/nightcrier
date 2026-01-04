# configuration Spec Delta

## MODIFIED Requirements

### Requirement: Single Source of Configuration Truth

The system SHALL NOT define default values for required configuration parameters in multiple locations.

#### Scenario: No deprecated helper methods
- **WHEN** checking storage backend type
- **THEN** code SHALL access `ObjectStorage.Type` directly
- **AND** deprecated helper methods like `IsAzureStorageEnabled()` SHALL NOT exist

_Note: This scenario documents removal of the deprecated `IsAzureStorageEnabled()` method. Code should access config fields directly rather than through deprecated wrapper methods._
